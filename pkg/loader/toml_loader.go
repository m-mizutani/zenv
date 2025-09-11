package loader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/zenv/pkg/model"
)

func NewTOMLLoader(path string) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, nil // File not found is acceptable
		}

		var config model.TOMLConfig
		if _, err := toml.DecodeFile(path, &config); err != nil {
			return nil, fmt.Errorf("failed to parse TOML file: %w", err)
		}

		var envVars []*model.EnvVar

		for key, value := range config {
			if err := value.Validate(); err != nil {
				return nil, fmt.Errorf("invalid configuration for %s: %w", key, err)
			}

			var envValue string
			var err error

			switch {
			case value.Value != nil:
				envValue = *value.Value
			case value.File != nil:
				envValue, err = readFile(*value.File)
				if err != nil {
					return nil, fmt.Errorf("failed to read file for %s: %w", key, err)
				}
			case value.Command != nil:
				envValue, err = executeCommand(*value.Command, value.Args)
				if err != nil {
					return nil, fmt.Errorf("failed to execute command for %s: %w", key, err)
				}
			}

			envVars = append(envVars, &model.EnvVar{
				Name:   key,
				Value:  envValue,
				Source: model.SourceTOML,
			})
		}

		return envVars, nil
	}
}

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path) // #nosec G304 - file path is user provided and expected
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func executeCommand(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
