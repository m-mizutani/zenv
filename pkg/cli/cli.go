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
	"github.com/urfave/cli/v3"
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
	var envFiles []string
	var tomlFiles []string
	var logLevel string

	app := &cli.Command{
		Name:  "zenv",
		Usage: "Environment variable loader and command executor",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "env",
				Aliases:     []string{"e"},
				Usage:       "Load environment variables from .env file",
				Destination: &envFiles,
			},
			&cli.StringSliceFlag{
				Name:        "toml",
				Aliases:     []string{"t"},
				Usage:       "Load environment variables from .toml file",
				Destination: &tomlFiles,
			},
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "Set log level (debug, info, warn, error)",
				Value:       "warn",
				Destination: &logLevel,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
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

			// Load .env files first and collect their variables
			var envLoaders []loader.LoadFunc
			for _, envFile := range envFiles {
				envLoaders = append(envLoaders, loader.NewDotEnvLoader(envFile))
			}
			if len(envFiles) == 0 {
				envLoaders = append(envLoaders, loader.NewDotEnvLoader(".env"))
			}

			// Execute .env loaders and collect results
			for _, loadFunc := range envLoaders {
				envVars, err := loadFunc(ctx)
				if err != nil {
					return goerr.Wrap(err, "failed to load .env file")
				}
				if envVars != nil {
					allExistingVars = append(allExistingVars, envVars...)
				}
			}

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
			loaders = append(loaders, envLoaders...)
			loaders = append(loaders, tomlLoaders...)

			// Create executor and usecase
			exec := executor.NewDefaultExecutor()
			uc := usecase.NewUseCase(loaders, exec)

			// Get command arguments (excluding program name and flags)
			args := cmd.Args().Slice()

			// If no command specified, force list mode
			if len(args) == 0 {
				args = []string{} // Force empty args to show environment variables
			}

			err := uc.Run(ctx, args)
			if err != nil {
				exitCode := model.GetExitCode(err)
				return model.WithExitCode(err, exitCode)
			}
			return nil
		},
	}

	return app.Run(ctx, args)
}
