package loader

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func NewTOMLLoader(path string) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)
		logger.Debug("loading TOML file", "path", path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			logger.Debug("TOML file not found", "path", path)
			return nil, nil // File not found is acceptable
		}

		var config model.TOMLConfig
		if _, err := toml.DecodeFile(path, &config); err != nil {
			logger.Error("failed to parse TOML file", "path", path, "error", err)
			return nil, goerr.Wrap(err, "failed to parse TOML file", goerr.V("path", path))
		}

		// Create unified resolver
		resolver := newUnifiedResolver(config)

		// Resolve all variables
		var envVars []*model.EnvVar
		for key, value := range config {
			if err := value.Validate(); err != nil {
				logger.Error("invalid TOML configuration", "key", key, "error", err)
				return nil, goerr.Wrap(err, "invalid configuration", goerr.V("key", key))
			}

			logger.Debug("resolving TOML variable", "key", key)
			resolvedValue, err := resolver.resolve(key)
			if err != nil {
				logger.Error("failed to resolve TOML variable", "key", key, "error", err)
				return nil, goerr.Wrap(err, "failed to resolve variable",
					goerr.V("key", key))
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceTOML,
			}
			envVars = append(envVars, envVar)
		}

		logger.Debug("loaded TOML file", "path", path, "variables", len(envVars))
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

// unifiedResolver handles resolution of all variable types with circular reference detection
type unifiedResolver struct {
	config       model.TOMLConfig
	resolvedVars map[string]string
	resolving    map[string]bool // Track variables currently being resolved
	systemEnvs   map[string]string
}

func newUnifiedResolver(config model.TOMLConfig) *unifiedResolver {
	// Cache system environment variables
	systemEnvs := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			systemEnvs[parts[0]] = parts[1]
		}
	}

	return &unifiedResolver{
		config:       config,
		resolvedVars: make(map[string]string),
		resolving:    make(map[string]bool),
		systemEnvs:   systemEnvs,
	}
}

func (r *unifiedResolver) resolve(key string) (string, error) {
	// Check if already resolved
	if value, exists := r.resolvedVars[key]; exists {
		return value, nil
	}

	// Check for circular reference
	if r.resolving[key] {
		return "", goerr.New("circular reference detected",
			goerr.V("key", key))
	}

	// Mark as currently resolving
	r.resolving[key] = true
	defer delete(r.resolving, key)

	// Get the configuration for this key
	config, exists := r.config[key]
	if !exists {
		// Not in TOML config, check system environment
		if value, exists := r.systemEnvs[key]; exists {
			r.resolvedVars[key] = value
			return value, nil
		}
		// Not found anywhere
		return "", nil
	}

	// Resolve based on type
	var resolvedValue string
	var err error

	switch {
	case config.Value != nil:
		resolvedValue = *config.Value

	case config.File != nil:
		resolvedValue, err = readFile(*config.File)
		if err != nil {
			return "", goerr.Wrap(err, "failed to read file",
				goerr.V("file", *config.File))
		}

	case config.Command != nil:
		resolvedValue, err = executeCommand(*config.Command, config.Args)
		if err != nil {
			return "", goerr.Wrap(err, "failed to execute command",
				goerr.V("command", *config.Command),
				goerr.V("args", config.Args))
		}

	case config.Alias != nil:
		// Recursively resolve the alias target
		resolvedValue, err = r.resolve(*config.Alias)
		if err != nil {
			return "", goerr.Wrap(err, "failed to resolve alias",
				goerr.V("alias", *config.Alias))
		}

	case config.Template != nil:
		// Build context for template
		context := make(map[string]string)
		for _, ref := range config.Refs {
			refValue, err := r.resolve(ref)
			if err != nil {
				return "", goerr.Wrap(err, "failed to resolve template reference",
					goerr.V("ref", ref))
			}
			context[ref] = refValue
		}

		// Parse and execute template
		tmpl, err := template.New("env").Parse(*config.Template)
		if err != nil {
			return "", goerr.Wrap(err, "failed to parse template",
				goerr.V("template", *config.Template))
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, context); err != nil {
			return "", goerr.Wrap(err, "failed to execute template",
				goerr.V("template", *config.Template),
				goerr.V("key", key))
		}

		resolvedValue = buf.String()
	}

	// Cache the resolved value
	r.resolvedVars[key] = resolvedValue
	return resolvedValue, nil
}
