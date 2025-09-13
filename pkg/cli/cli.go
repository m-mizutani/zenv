package cli

import (
	"context"
	"io"
	"log/slog"
	"os"
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

// ParseLogLevel parses a string log level to slog.Level
func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO", "":
		return slog.LevelInfo
	case "warn", "warning", "WARN", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelWarn // Default to warn as specified
	}
}

func Run(ctx context.Context, args []string) error {
	// Create parser and configure options
	parser := NewParser()

	err := parser.AddOption(Option{
		Name:         "env",
		Aliases:      []string{"e"},
		Usage:        "Load environment variables from .env file",
		DefaultValue: ".env",
		IsSlice:      true,
	})
	if err != nil {
		return goerr.Wrap(err, "failed to add env option")
	}

	err = parser.AddOption(Option{
		Name:    "toml",
		Aliases: []string{"t"},
		Usage:   "Load environment variables from .toml file",
		IsSlice: true,
	})
	if err != nil {
		return goerr.Wrap(err, "failed to add toml option")
	}

	err = parser.AddOption(Option{
		Name:         "log-level",
		Aliases:      []string{"l"},
		Usage:        "Set log level (debug, info, warn, error)",
		DefaultValue: "warn",
	})
	if err != nil {
		return goerr.Wrap(err, "failed to add log-level option")
	}

	// Check for help flag first
	for _, arg := range args[1:] {
		if arg == "-h" || arg == "--help" {
			os.Stdout.WriteString("Usage: zenv [options] <command> [args...]\n\n")
			os.Stdout.WriteString("Options:\n")
			os.Stdout.WriteString(parser.Help() + "\n")
			return nil
		}
		// Stop checking after first non-option
		if !strings.HasPrefix(arg, "-") {
			break
		}
	}

	// Parse arguments
	result, err := parser.Parse(ctx, args[1:]) // Skip program name
	if err != nil {
		// Show help message with error
		os.Stderr.WriteString("\nUsage: zenv [options] <command> [args...]\n\n")
		os.Stderr.WriteString("Options:\n")
		os.Stderr.WriteString(parser.Help() + "\n")
		return err
	}

	// Extract parsed values
	envFiles := result.Options["env"].StringSlice()
	tomlFiles := result.Options["toml"].StringSlice()
	logLevel := result.Options["log-level"].String()
	commandArgs := result.Args

	// Create logger based on log-level flag
	level := ParseLogLevel(logLevel)
	logger := NewLogger(level, os.Stderr)

	// Set logger in context for propagation
	ctx = ctxlog.With(ctx, logger)

	// Collect environment variables in order for TOML loader reference
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

	// Load .env files once and collect their variables
	var envLoaders []loader.LoadFunc
	for _, envFile := range envFiles {
		envLoaders = append(envLoaders, loader.NewDotEnvLoader(envFile))
	}
	if len(envFiles) == 0 {
		envLoaders = append(envLoaders, loader.NewDotEnvLoader(".env"))
	}

	// Execute .env loaders once and collect results
	var loadedDotEnvVars []*model.EnvVar
	for _, loadFunc := range envLoaders {
		envVars, err := loadFunc(ctx)
		if err != nil {
			return goerr.Wrap(err, "failed to load .env file")
		}
		if envVars != nil {
			loadedDotEnvVars = append(loadedDotEnvVars, envVars...)
		}
	}
	allExistingVars = append(allExistingVars, loadedDotEnvVars...)

	// Now create TOML loaders with all existing variables
	var tomlLoaders []loader.LoadFunc
	for _, tomlFile := range tomlFiles {
		tomlLoaders = append(tomlLoaders, loader.NewTOMLLoader(tomlFile, allExistingVars))
	}
	if len(tomlFiles) == 0 {
		tomlLoaders = append(tomlLoaders, loader.NewTOMLLoader(".env.toml", allExistingVars))
	}

	// Combine all loaders for the usecase
	var loaders []loader.LoadFunc
	// Use an in-memory loader for .env vars to avoid reading files twice
	loaders = append(loaders, func(ctx context.Context) ([]*model.EnvVar, error) {
		return loadedDotEnvVars, nil
	})
	loaders = append(loaders, tomlLoaders...)

	// Create executor and usecase
	exec := executor.NewDefaultExecutor()
	uc := usecase.NewUseCase(loaders, exec)

	// If no command specified, force list mode
	if len(commandArgs) == 0 {
		commandArgs = []string{} // Force empty args to show environment variables
	}

	if err := uc.Run(ctx, commandArgs); err != nil {
		exitCode := model.GetExitCode(err)
		return model.WithExitCode(err, exitCode)
	}
	return nil
}
