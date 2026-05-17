package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/m-mizutani/clog"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"github.com/m-mizutani/zenv/v2/pkg/usecase"
	"golang.org/x/term"
)

// checkProfileExists verifies that the requested profile name is defined in
// at least one of the supplied configuration files. The check is silent when
// the profile is found; otherwise it returns a *model.ProfileNotFoundError
// describing what was searched and which alternatives exist.
//
// When mustExist is true (caller passed --config explicitly), a missing file
// surfaces as the underlying *model.ConfigFileError so the user sees a
// targeted "missing file" error rather than the indirect "profile not found".
// Only paths that actually contributed configuration are recorded in
// ProfileNotFoundError.Paths, which lets the hint system distinguish
// "configuration loaded but profile not defined" from "nothing was loaded".
func checkProfileExists(ctx context.Context, profile string, paths []string, mustExist bool) error {
	combined := make(map[string]struct{})
	var foundPaths []string
	for _, p := range paths {
		names, err := loader.CollectProfileNames(ctx, p, mustExist)
		if err != nil {
			return err
		}
		if names == nil {
			continue
		}
		foundPaths = append(foundPaths, p)
		for name := range names {
			combined[name] = struct{}{}
		}
	}
	if _, ok := combined[profile]; ok {
		return nil
	}

	available := make([]string, 0, len(combined))
	for name := range combined {
		available = append(available, name)
	}
	sort.Strings(available)

	return &model.ProfileNotFoundError{
		Profile:   profile,
		Available: available,
		Paths:     foundPaths,
	}
}

// newConfigLoader picks the appropriate loader based on the file extension.
// Files ending in .hcl use the HCL loader; everything else falls back to YAML.
// mustExist=true is set when the path comes from an explicit --config flag, so
// the loader fails fast on a missing file; default-discovery paths pass false.
func newConfigLoader(path, profile string, mustExist bool, existingVars []*model.EnvVar) loader.LoadFunc {
	if strings.EqualFold(filepath.Ext(path), ".hcl") {
		return loader.NewHCLLoaderWithProfile(path, profile, mustExist, existingVars)
	}
	return loader.NewYAMLLoaderWithProfile(path, profile, mustExist, existingVars)
}

// Format represents the log output format
type Format int

const (
	FormatAuto Format = iota
	FormatConsole
	FormatJSON
)

// NewLogger creates a new slog.Logger with automatic format detection
// If output is a terminal, use clog for colored console output
// Otherwise, use JSON format for structured logging
func NewLogger(level slog.Level, w io.Writer) *slog.Logger {
	return NewLoggerWithFormat(level, w, FormatAuto)
}

// NewLoggerWithFormat creates a new slog.Logger with specified format
func NewLoggerWithFormat(level slog.Level, w io.Writer, format Format) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	useConsole := format == FormatConsole
	if format == FormatAuto {
		isTerminal := false
		if f, ok := w.(*os.File); ok {
			isTerminal = term.IsTerminal(int(f.Fd()))
		}
		useConsole = isTerminal
	}

	var handler slog.Handler
	if useConsole {
		// Console output with colors
		handler = clog.New(
			clog.WithWriter(w),
			clog.WithLevel(level),
			clog.WithTimeFmt("15:04:05"),
			clog.WithSource(false),
		)
	} else {
		// JSON output for non-terminal (logs, CI/CD, etc.)
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: level,
		})
	}

	return slog.New(handler)
}

// isTruthyEnv reports whether the given environment variable string should be
// treated as enabling a boolean flag. Accepts the common conventions ("1",
// "true", "yes", "on") case-insensitively. An empty string is false.
func isTruthyEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// allowedLogLevels lists the user-facing log level names accepted by
// ParseLogLevel. Kept here so the error message and the parser stay in sync.
var allowedLogLevels = []string{"debug", "info", "warn", "error"}

// ParseLogLevel parses a string log level to slog.Level.
//
// Unknown values (including the empty string) are reported as
// model.InvalidLogLevelError so the caller can surface a precise error to the
// user instead of silently falling back to a default. The CLI parser supplies
// "warn" as the default when --log-level is omitted, so empty input here means
// the user explicitly passed an empty value, which is also invalid.
func ParseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "info", "INFO":
		return slog.LevelInfo, nil
	case "warn", "warning", "WARN", "WARNING":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	default:
		return 0, &model.InvalidLogLevelError{
			Value:   level,
			Allowed: allowedLogLevels,
		}
	}
}

func Run(ctx context.Context, args []string) error {
	// Create parser with options
	parser, err := NewParser([]Option{
		{
			Name:      "help",
			Aliases:   []string{"h"},
			Usage:     "Show help message",
			IsBoolean: true,
		},
		{
			Name:    "env",
			Aliases: []string{"e"},
			Usage:   "Load environment variables from .env file",
			IsSlice: true,
		},
		{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load environment variables from .yaml file",
			IsSlice: true,
		},
		{
			Name:    "profile",
			Aliases: []string{"p"},
			Usage:   "Select profile from YAML configuration",
		},
		{
			Name:         "log-level",
			Aliases:      []string{"l"},
			Usage:        "Set log level (debug, info, warn, error)",
			DefaultValue: "warn",
		},
		{
			Name:      "template",
			Aliases:   []string{"t"},
			Usage:     "Enable template expansion for command arguments using text/template syntax",
			IsBoolean: true,
		},
		{
			Name:      "redact",
			Usage:     "Redact secret values from the child process's stdout/stderr (also enabled by ZENV_REDACT=1)",
			IsBoolean: true,
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to create parser")
	}

	// Parse arguments
	result, err := parser.Parse(ctx, args[1:]) // Skip program name
	if err != nil {
		// Check if help was requested
		if errors.Is(err, ErrHelpRequested) {
			_, _ = os.Stdout.WriteString("Usage: zenv [options] <command> [args...]\n\n")
			_, _ = os.Stdout.WriteString("Options:\n")
			_, _ = os.Stdout.WriteString(parser.Help() + "\n")
			return nil
		}
		// Show help message with error
		_, _ = os.Stderr.WriteString("\nUsage: zenv [options] <command> [args...]\n\n")
		_, _ = os.Stderr.WriteString("Options:\n")
		_, _ = os.Stderr.WriteString(parser.Help() + "\n")
		return err
	}

	// Extract parsed values
	envFiles := result.Options["env"].StringSlice()
	configFiles := result.Options["config"].StringSlice()
	logLevel := result.Options["log-level"].String()
	profile := result.Options["profile"].String()
	enableTemplate := result.Options["template"].IsSet()
	redactEnabled := result.Options["redact"].IsSet() || isTruthyEnv(os.Getenv("ZENV_REDACT"))
	commandArgs := result.Args

	// Create logger based on log-level flag. An invalid level is reported up
	// the chain so the user sees the standard FormatError output, just like
	// every other start-up failure.
	level, err := ParseLogLevel(logLevel)
	if err != nil {
		return err
	}
	logger := NewLogger(level, os.Stderr)

	// Set logger in context for propagation
	ctx = ctxlog.With(ctx, logger)

	// Resolve the configuration paths that should drive both the profile
	// preflight and the actual loader chain.
	var configPaths []string
	configExplicit := len(configFiles) > 0
	if configExplicit {
		configPaths = configFiles
	} else {
		// Default path resolution: prefer .env.hcl if present (do not merge with YAML).
		if hclPath := loader.FindDefaultHCLPath(); hclPath != "" {
			configPaths = []string{hclPath}
		} else {
			configPaths = []string{loader.ResolveDefaultYAMLPath()}
		}
	}

	// Pre-flight: when --profile is set, ensure the requested profile name is
	// defined somewhere in the configuration files we are about to load. We do
	// this before running any loader so the user doesn't see partial work
	// (e.g. a successful command resolution) for an obviously wrong flag.
	if profile != "" {
		if err := checkProfileExists(ctx, profile, configPaths, configExplicit); err != nil {
			return err
		}
	}

	// Collect environment variables in order for YAML loader reference
	var allExistingVars []*model.EnvVar

	// First, collect system environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			allExistingVars = append(allExistingVars, &model.EnvVar{
				Name:   parts[0],
				Value:  parts[1],
				Source: model.SourceSystem,
			})
		}
	}

	// Load .env files once and collect their variables. Explicitly passed
	// paths (-e/--env) must exist; the default .env probe stays tolerant.
	var envLoaders []loader.LoadFunc
	for _, envFile := range envFiles {
		envLoaders = append(envLoaders, loader.NewDotEnvLoader(envFile, true))
	}
	if len(envFiles) == 0 {
		envLoaders = append(envLoaders, loader.NewDotEnvLoader(loader.ResolveDefaultDotEnvPath(), false))
	}

	// Execute .env loaders once and collect results
	var loadedDotEnvVars []*model.EnvVar
	for _, loadFunc := range envLoaders {
		envVars, err := loadFunc(ctx)
		if err != nil {
			return err
		}
		if envVars != nil {
			loadedDotEnvVars = append(loadedDotEnvVars, envVars...)
		}
	}
	allExistingVars = append(allExistingVars, loadedDotEnvVars...)

	// Build config loaders for the same paths we used in the preflight.
	// Explicit --config paths must exist; default-discovery paths stay tolerant.
	var configLoaders []loader.LoadFunc
	for _, configPath := range configPaths {
		configLoaders = append(configLoaders, newConfigLoader(configPath, profile, configExplicit, allExistingVars))
	}

	// Combine all loaders for the usecase
	var loaders []loader.LoadFunc
	// Use an in-memory loader for .env vars to avoid reading files twice
	loaders = append(loaders, func(ctx context.Context) ([]*model.EnvVar, error) {
		return loadedDotEnvVars, nil
	})
	loaders = append(loaders, configLoaders...)

	// Create executor and usecase
	exec := executor.NewDefaultExecutor(executor.Options{Redact: redactEnabled})
	uc := usecase.NewUseCase(loaders, exec)
	uc.EnableTemplate = enableTemplate

	// If no command specified, force list mode
	if len(commandArgs) == 0 {
		commandArgs = []string{} // Force empty args to show environment variables
	}

	if err := uc.Run(ctx, commandArgs); err != nil {
		// ExecutorError = the target command actually ran and exited non-zero.
		// Its own stderr is already on the user's terminal, so we must NOT
		// print another error block on top of it.
		//
		// Every other error path (CommandLaunchError when the child never
		// launched, config-load failures, variable-resolution failures, ...)
		// has produced no user-facing output yet, so it goes through the
		// FormatError pipeline.
		if !model.IsExecutorError(err) {
			verbose := level <= slog.LevelDebug
			useColor := false
			if f, ok := any(os.Stderr).(*os.File); ok {
				useColor = term.IsTerminal(int(f.Fd()))
			}
			_, _ = fmt.Fprintln(os.Stderr, FormatError(err, verbose, useColor))
		}
		return err
	}
	return nil
}
