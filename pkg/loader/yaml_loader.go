package loader

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"gopkg.in/yaml.v3"
)

func NewYAMLLoader(path string, existingVars ...[]*model.EnvVar) LoadFunc {
	return NewYAMLLoaderWithProfile(path, "", existingVars...)
}

func NewYAMLLoaderWithProfile(path string, profile string, existingVars ...[]*model.EnvVar) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)
		logger.Debug("loading YAML file", "path", path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			logger.Debug("YAML file not found", "path", path)
			return nil, nil // File not found is acceptable
		}

		data, err := os.ReadFile(path) // #nosec G304 - file path is user provided and expected
		if err != nil {
			logger.Error("failed to read YAML file", "path", path, "error", err)
			return nil, goerr.Wrap(err, "failed to read YAML file", goerr.V("path", path))
		}

		var config model.YAMLConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			logger.Error("failed to parse YAML file", "path", path, "error", err)
			return nil, goerr.Wrap(err, "failed to parse YAML file", goerr.V("path", path))
		}

		// Merge existing variables if provided
		var allExistingVars []*model.EnvVar
		for _, vars := range existingVars {
			allExistingVars = append(allExistingVars, vars...)
		}

		// Create unified resolver with existing variables
		resolver := newYAMLUnifiedResolverWithProfileAndVars(config, profile, allExistingVars)

		// Resolve all variables
		var envVars []*model.EnvVar
		for key, value := range config {
			// Get value for the specified profile
			effectiveValue := value.GetValueForProfile(profile)

			// Skip if the value is not defined for this profile (nil) or is explicitly unset (empty)
			if effectiveValue == nil || effectiveValue.IsEmpty() {
				logger.Debug("skipping variable (unset or not defined in profile)", "key", key, "profile", profile)
				continue
			}

			if err := effectiveValue.Validate(); err != nil {
				logger.Error("invalid YAML configuration", "key", key, "error", err)
				return nil, goerr.Wrap(err, "invalid configuration", goerr.V("key", key))
			}

			logger.Debug("resolving YAML variable", "key", key)
			resolvedValue, err := resolver.resolveWithValue(key, effectiveValue)
			if err != nil {
				logger.Error("failed to resolve YAML variable", "key", key, "error", err)
				return nil, goerr.Wrap(err, "failed to resolve variable",
					goerr.V("key", key))
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceYAML,
			}
			envVars = append(envVars, envVar)
		}

		logger.Debug("loaded YAML file", "path", path, "variables", len(envVars))
		return envVars, nil
	}
}

func readYAMLFile(path string) (string, error) {
	content, err := os.ReadFile(path) // #nosec G304 - file path is user provided and expected
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func executeYAMLCommand(command []string) (string, error) {
	if len(command) == 0 {
		return "", goerr.New("command is empty")
	}
	cmd := exec.Command(command[0], command[1:]...) // #nosec G204 - command is from user-provided YAML config, which is expected
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// yamlUnifiedResolver handles resolution of all variable types with circular reference detection
type yamlUnifiedResolver struct {
	config       model.YAMLConfig
	profile      string
	resolvedVars map[string]string
	resolving    map[string]bool   // Track variables currently being resolved
	externalVars map[string]string // Variables from .env files, system environment, and other sources
}

func newYAMLUnifiedResolverWithProfileAndVars(config model.YAMLConfig, profile string, existingVars []*model.EnvVar) *yamlUnifiedResolver {
	// Cache all external variables (system environment + .env files, etc.)
	externalVars := make(map[string]string)

	// First add system environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			externalVars[parts[0]] = parts[1]
		}
	}

	// Then add existing variables (from .env files, etc.) - these can override system vars
	for _, envVar := range existingVars {
		if envVar != nil {
			externalVars[envVar.Name] = envVar.Value
		}
	}

	return &yamlUnifiedResolver{
		config:       config,
		profile:      profile,
		resolvedVars: make(map[string]string),
		resolving:    make(map[string]bool),
		externalVars: externalVars,
	}
}

// buildTemplateContext resolves all refs and builds a context map for template execution
func (r *yamlUnifiedResolver) buildTemplateContext(refs []string) (map[string]string, error) {
	context := make(map[string]string)
	for _, ref := range refs {
		refValue, err := r.resolve(ref)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to resolve reference",
				goerr.V("ref", ref))
		}
		context[ref] = refValue
	}
	return context, nil
}

func (r *yamlUnifiedResolver) resolve(key string) (string, error) {
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
		// Not in YAML config, check external variables (which includes system vars)
		if value, exists := r.externalVars[key]; exists {
			r.resolvedVars[key] = value
			return value, nil
		}
		// Not found anywhere - return error for missing variable
		return "", goerr.New("variable not found",
			goerr.V("key", key))
	}

	// Get effective value considering profile
	effectiveValue := config.GetValueForProfile(r.profile)
	return r.resolveWithValue(key, effectiveValue)
}

func (r *yamlUnifiedResolver) resolveWithValue(key string, config *model.YAMLValue) (string, error) {
	if config == nil {
		return "", goerr.New("nil configuration for key",
			goerr.V("key", key))
	}

	// Resolve based on type
	var resolvedValue string
	var err error

	switch {
	case config.Value != nil:
		// If refs are present, treat value as a template
		if len(config.Refs) > 0 {
			// Build context for template
			context, err := r.buildTemplateContext(config.Refs)
			if err != nil {
				return "", goerr.Wrap(err, "failed to build template context")
			}

			// Parse and execute template
			tmpl, err := template.New("env").Parse(*config.Value)
			if err != nil {
				return "", goerr.Wrap(err, "failed to parse value template",
					goerr.V("value", *config.Value))
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, context); err != nil {
				return "", goerr.Wrap(err, "failed to execute value template",
					goerr.V("value", *config.Value),
					goerr.V("key", key))
			}

			resolvedValue = buf.String()
		} else {
			// No refs, use value as-is
			resolvedValue = *config.Value
		}

	case config.File != nil:
		resolvedValue, err = readYAMLFile(*config.File)
		if err != nil {
			return "", goerr.Wrap(err, "failed to read file",
				goerr.V("file", *config.File))
		}

	case len(config.Command) > 0:
		// Resolve command with optional refs
		commandToExecute := config.Command

		// If refs are present, resolve them and apply templates to command elements
		if len(config.Refs) > 0 {
			// Build context for template
			context, err := r.buildTemplateContext(config.Refs)
			if err != nil {
				return "", goerr.Wrap(err, "failed to build command template context")
			}

			// Apply template to each command element
			resolvedCommand := make([]string, len(config.Command))
			tmpl := template.New("cmd")
			for i, cmdElement := range config.Command {
				parsedTmpl, err := tmpl.Parse(cmdElement)
				if err != nil {
					return "", goerr.Wrap(err, "failed to parse command template",
						goerr.V("element", cmdElement))
				}

				var buf bytes.Buffer
				if err := parsedTmpl.Execute(&buf, context); err != nil {
					return "", goerr.Wrap(err, "failed to execute command template",
						goerr.V("element", cmdElement))
				}
				resolvedCommand[i] = buf.String()
			}
			commandToExecute = resolvedCommand
		}

		resolvedValue, err = executeYAMLCommand(commandToExecute)
		if err != nil {
			return "", goerr.Wrap(err, "failed to execute command",
				goerr.V("command", commandToExecute))
		}

	case config.Alias != nil:
		// Recursively resolve the alias target
		resolvedValue, err = r.resolve(*config.Alias)
		if err != nil {
			return "", goerr.Wrap(err, "failed to resolve alias",
				goerr.V("alias", *config.Alias))
		}
	}

	// Cache the resolved value
	r.resolvedVars[key] = resolvedValue
	return resolvedValue, nil
}
