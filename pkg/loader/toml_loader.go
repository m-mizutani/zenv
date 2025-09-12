package loader

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"text/template"

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
			return nil, goerr.Wrap(err, "failed to parse TOML file", goerr.V("path", path))
		}

		var envVars []*model.EnvVar
		aliasResolver := newAliasResolver(config)
		templateResolver := newTemplateResolver(config)

		// First, collect all non-alias and non-template values
		for key, value := range config {
			if err := value.Validate(); err != nil {
				return nil, goerr.Wrap(err, "invalid configuration", goerr.V("key", key))
			}

			if value.Alias != nil || value.Template != nil {
				// Skip alias and template values for now, will process them later
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
					return nil, goerr.Wrap(err, "failed to read file",
						goerr.V("key", key),
						goerr.V("file", *value.File))
				}
			case value.Command != nil:
				envValue, err = executeCommand(*value.Command, value.Args)
				if err != nil {
					return nil, goerr.Wrap(err, "failed to execute command",
						goerr.V("key", key),
						goerr.V("command", *value.Command),
						goerr.V("args", value.Args))
				}
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  envValue,
				Source: model.SourceTOML,
			}
			envVars = append(envVars, envVar)
			aliasResolver.addResolvedVar(key, envValue)
			templateResolver.addResolvedVar(key, envValue)
		}

		// Then, resolve all alias values
		for key, value := range config {
			if value.Alias == nil {
				continue
			}

			resolvedValue, err := aliasResolver.resolve(*value.Alias, key)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to resolve alias",
					goerr.V("key", key),
					goerr.V("alias", *value.Alias))
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceTOML,
			}
			envVars = append(envVars, envVar)
			templateResolver.addResolvedVar(key, resolvedValue)
		}

		// Finally, resolve all template values
		for key, value := range config {
			if value.Template == nil {
				continue
			}

			resolvedValue, err := templateResolver.resolve(*value.Template, value.Refs, key)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to resolve template",
					goerr.V("key", key),
					goerr.V("template", *value.Template),
					goerr.V("refs", value.Refs))
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
		return "", goerr.New("circular alias reference detected",
			goerr.V("target", aliasTarget),
			goerr.V("visited", visited))
	}

	// First, check if it's already resolved from TOML config
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

	// If not found in TOML, check system environment variables
	if value, ok := os.LookupEnv(aliasTarget); ok {
		return value, nil
	}

	// If not found anywhere, return empty string
	return "", nil
}

// templateResolver handles template resolution with circular reference detection
type templateResolver struct {
	config       model.TOMLConfig
	resolvedVars map[string]string
	systemEnvs   map[string]string // Cache system environment variables
}

func newTemplateResolver(config model.TOMLConfig) *templateResolver {
	// Cache system environment variables
	systemEnvs := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			systemEnvs[parts[0]] = parts[1]
		}
	}

	return &templateResolver{
		config:       config,
		resolvedVars: make(map[string]string),
		systemEnvs:   systemEnvs,
	}
}

func (r *templateResolver) addResolvedVar(key string, value string) {
	r.resolvedVars[key] = value
}

func (r *templateResolver) resolve(templateStr string, refs []string, currentKey string) (string, error) {
	return r.resolveWithVisited(templateStr, refs, currentKey, make(map[string]bool))
}

func (r *templateResolver) resolveWithVisited(templateStr string, refs []string, currentKey string, visited map[string]bool) (string, error) {
	// Check for circular reference
	if visited[currentKey] {
		return "", goerr.New("circular template reference detected",
			goerr.V("key", currentKey),
			goerr.V("visited", visited))
	}
	visited[currentKey] = true

	// Build the context for the template
	context := make(map[string]string)

	for _, ref := range refs {
		value, err := r.resolveRef(ref, visited)
		if err != nil {
			return "", goerr.Wrap(err, "failed to resolve reference",
				goerr.V("ref", ref),
				goerr.V("key", currentKey))
		}
		context[ref] = value
	}

	// Parse and execute the template
	tmpl, err := template.New("env").Parse(templateStr)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template",
			goerr.V("template", templateStr),
			goerr.V("key", currentKey))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", goerr.Wrap(err, "failed to execute template",
			goerr.V("template", templateStr),
			goerr.V("key", currentKey),
			goerr.V("context", context))
	}

	return buf.String(), nil
}

func (r *templateResolver) resolveRef(ref string, visited map[string]bool) (string, error) {
	// Check for circular reference
	if visited[ref] {
		return "", goerr.New("circular template reference detected",
			goerr.V("ref", ref),
			goerr.V("visited", visited))
	}

	// First, check if it's already resolved from TOML config
	if value, exists := r.resolvedVars[ref]; exists {
		return value, nil
	}

	// Check if the ref exists in config and needs resolution
	if refConfig, exists := r.config[ref]; exists {
		if refConfig.Template != nil {
			// Use resolveWithVisited to properly track circular references
			return r.resolveWithVisited(*refConfig.Template, refConfig.Refs, ref, visited)
		}
		// If it's not a template but exists in config, it should have been resolved already
		// This shouldn't happen in normal flow
	}

	// If not found in TOML, check system environment variables
	if value, exists := r.systemEnvs[ref]; exists {
		return value, nil
	}

	// If not found anywhere, return empty string (not an error)
	return "", nil
}
