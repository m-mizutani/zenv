package loader

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func NewDotEnvLoader(path string) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)
		logger.Debug("loading .env file", "path", path)

		file, err := os.Open(filepath.Clean(path))
		if err != nil {
			if os.IsNotExist(err) {
				logger.Debug(".env file not found", "path", path)
				return nil, nil // File not found is acceptable
			}
			logger.Error("failed to open .env file", "path", path, "error", err)
			return nil, goerr.Wrap(err, "failed to open .env file")
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
				return nil, goerr.New("invalid format in .env file",
					goerr.V("line", lineNumber),
					goerr.V("content", line),
				)
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
			logger.Error("failed to read .env file", "path", path, "error", err)
			return nil, goerr.Wrap(err, "failed to read .env file")
		}

		logger.Debug("loaded .env file", "path", path, "variables", len(envVars))
		return envVars, nil
	}
}
