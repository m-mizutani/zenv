package loader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func NewDotEnvLoader(path string) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		file, err := os.Open(path) // #nosec G304 - file path is user provided and expected
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil // File not found is acceptable
			}
			return nil, fmt.Errorf("failed to open .env file: %w", err)
		}
		defer file.Close()

		var envVars []*model.EnvVar
		scanner := bufio.NewScanner(file)
		lineNumber := 0

		for scanner.Scan() {
			lineNumber++
			line := strings.TrimSpace(scanner.Text())

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Parse KEY=VALUE
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid format at line %d: %s", lineNumber, line)
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}

			envVars = append(envVars, &model.EnvVar{
				Name:   key,
				Value:  value,
				Source: model.SourceDotEnv,
			})
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read .env file: %w", err)
		}

		return envVars, nil
	}
}
