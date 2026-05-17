package loader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// NewDotEnvLoader builds a loader for a single .env file. When mustExist is
// true, a missing file is reported as an error; otherwise it is silently
// skipped (used for the default .env path).
func NewDotEnvLoader(path string, mustExist bool) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)
		logger.Debug("loading .env file", "path", path)

		file, err := os.Open(filepath.Clean(path))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if mustExist {
					return nil, &model.ConfigFileError{
						Path:   path,
						Format: model.FormatDotEnv,
						Reason: model.ReasonNotFound,
						Cause:  err,
					}
				}
				logger.Debug(".env file not found", "path", path)
				return nil, nil
			}
			return nil, &model.ConfigFileError{
				Path:   path,
				Format: model.FormatDotEnv,
				Reason: model.ReasonNotReadable,
				Cause:  err,
			}
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
				return nil, &model.ConfigFileError{
					Path:   path,
					Format: model.FormatDotEnv,
					Reason: model.ReasonParseError,
					Detail: fmt.Sprintf("line %d: %q is not in KEY=VALUE format", lineNumber, line),
				}
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
			return nil, &model.ConfigFileError{
				Path:   path,
				Format: model.FormatDotEnv,
				Reason: model.ReasonNotReadable,
				Cause:  err,
			}
		}

		logger.Debug("loaded .env file", "path", path, "variables", len(envVars))
		return envVars, nil
	}
}
