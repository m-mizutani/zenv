package usecase

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

type UseCase struct {
	Loaders  []loader.LoadFunc
	Executor executor.ExecuteFunc
}

func NewUseCase(loaders []loader.LoadFunc, exec executor.ExecuteFunc) *UseCase {
	return &UseCase{
		Loaders:  loaders,
		Executor: exec,
	}
}

func (uc *UseCase) Run(ctx context.Context, args []string) error {
	logger := ctxlog.From(ctx)
	logger.Debug("starting zenv run", "args", args)

	// Parse inline environment variables and command
	inlineEnvVars, command, commandArgs := parseInlineEnvVars(args)
	logger.Debug("parsed arguments", "inline_vars", len(inlineEnvVars), "command", command, "command_args", commandArgs)

	// Load environment variables from all loaders
	var allEnvVars []*model.EnvVar

	// Add system environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			allEnvVars = append(allEnvVars, &model.EnvVar{
				Name:   parts[0],
				Value:  parts[1],
				Source: model.SourceSystem,
			})
		}
	}

	// Load from file loaders
	for _, loadFunc := range uc.Loaders {
		envVars, err := loadFunc(ctx)
		if err != nil {
			return goerr.Wrap(err, "failed to load environment variables")
		}
		allEnvVars = append(allEnvVars, envVars...)
	}

	// Add inline environment variables
	allEnvVars = append(allEnvVars, inlineEnvVars...)
	logger.Debug("loaded environment variables", "total", len(allEnvVars))

	// Merge environment variables (later sources override earlier ones)
	mergedEnvVars := mergeEnvVars(allEnvVars)
	logger.Debug("merged environment variables", "final_count", len(mergedEnvVars))

	// If no command is specified, show environment variables
	if command == "" {
		logger.Info("displaying environment variables", "count", len(mergedEnvVars))
		showEnvVars(mergedEnvVars)
		return nil
	}

	// Execute command with environment variables
	logger.Info("executing command", "command", command, "args", commandArgs, "env_vars", len(mergedEnvVars))
	err := uc.Executor(ctx, command, commandArgs, mergedEnvVars)
	if err != nil {
		return err // Return the error with embedded exit code
	}

	logger.Debug("command completed successfully")
	return nil
}

func parseInlineEnvVars(args []string) ([]*model.EnvVar, string, []string) {
	var inlineEnvVars []*model.EnvVar
	commandStart := -1

	for i, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				inlineEnvVars = append(inlineEnvVars, &model.EnvVar{
					Name:   parts[0],
					Value:  parts[1],
					Source: model.SourceInline,
				})
				continue
			}
		}
		commandStart = i
		break
	}

	if commandStart == -1 || commandStart >= len(args) {
		return inlineEnvVars, "", nil
	}

	return inlineEnvVars, args[commandStart], args[commandStart+1:]
}

func mergeEnvVars(envVars []*model.EnvVar) []*model.EnvVar {
	envMap := make(map[string]*model.EnvVar)

	for _, envVar := range envVars {
		envMap[envVar.Name] = envVar
	}

	var result []*model.EnvVar
	for _, envVar := range envMap {
		result = append(result, envVar)
	}

	return result
}

func showEnvVars(envVars []*model.EnvVar) {
	for _, envVar := range envVars {
		var sourceStr string
		switch envVar.Source {
		case model.SourceSystem:
			sourceStr = "system"
		case model.SourceDotEnv:
			sourceStr = ".env"
		case model.SourceTOML:
			sourceStr = ".toml"
		case model.SourceInline:
			sourceStr = "inline"
		}
		fmt.Printf("%s=%s [%s]\n", envVar.Name, envVar.Value, sourceStr)
	}
}
