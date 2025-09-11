package loader

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func NewTOMLLoader(path string) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, nil // File not found is acceptable
		}

		var config model.TOMLConfig
		if _, err := toml.DecodeFile(path, &config); err != nil {
			return nil, goerr.Wrap(err, "failed to parse TOML file")
		}

		var envVars []*model.EnvVar
		aliasResolver := newAliasResolver(config)

		// First, collect all non-alias values
		for key, value := range config {
			if err := value.Validate(); err != nil {
				return nil, goerr.Wrap(err, "invalid configuration for "+key)
			}

			if value.Alias != nil {
				// Skip alias values for now, will process them later
				continue
			}

			var envValue string
			var err error

			switch {
			case value.Value != nil:
				envValue = *value.Value
			case value.File != nil:
				envValue, err = readFile(*value.File)
				if err != nil {
					return nil, goerr.Wrap(err, "failed to read file for "+key)
				}
			case value.Command != nil:
				envValue, err = executeCommand(*value.Command, value.Args)
				if err != nil {
					return nil, goerr.Wrap(err, "failed to execute command for "+key)
				}
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  envValue,
				Source: model.SourceTOML,
			}
			envVars = append(envVars, envVar)
			aliasResolver.addResolvedVar(key, envValue)
		}

		// Then, resolve all alias values
		for key, value := range config {
			if value.Alias == nil {
				continue
			}

			resolvedValue, err := aliasResolver.resolve(*value.Alias, key)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to resolve alias for "+key)
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceTOML,
			}
			envVars = append(envVars, envVar)
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

// aliasResolver handles alias resolution with circular reference detection
type aliasResolver struct {
	config       model.TOMLConfig
	resolvedVars map[string]string
}

func newAliasResolver(config model.TOMLConfig) *aliasResolver {
	return &aliasResolver{
		config:       config,
		resolvedVars: make(map[string]string),
	}
}

func (r *aliasResolver) addResolvedVar(key string, value string) {
	r.resolvedVars[key] = value
}

func (r *aliasResolver) resolve(aliasTarget string, currentKey string) (string, error) {
	// Track visited keys to detect circular references
	visited := make(map[string]bool)
	visited[currentKey] = true // Mark the current key as visited to prevent self-reference
	return r.resolveWithVisited(aliasTarget, visited)
}

func (r *aliasResolver) resolveWithVisited(aliasTarget string, visited map[string]bool) (string, error) {
	// Check for circular reference
	if visited[aliasTarget] {
		return "", goerr.New("circular alias reference detected")
	}

	// First, check system environment variables
	if value := os.Getenv(aliasTarget); value != "" {
		return value, nil
	}

	// Check if it's already resolved
	if value, exists := r.resolvedVars[aliasTarget]; exists {
		return value, nil
	}

	// Check if the target exists in config and is an alias itself
	if targetConfig, exists := r.config[aliasTarget]; exists {
		if targetConfig.Alias != nil {
			// Mark this target as visited and recursively resolve
			visited[aliasTarget] = true
			return r.resolveWithVisited(*targetConfig.Alias, visited)
		}
		// If it's not an alias but exists in config, it should have been resolved already
		// This shouldn't happen in normal flow, but return empty string for safety
	}

	// If not found, return empty string (don't error out)
	return "", nil
}
