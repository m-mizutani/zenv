package loader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"gopkg.in/yaml.v3"
)

// stderrLimit caps how much of a failed command's stderr we retain on
// CommandExecError. The tail is kept so the most relevant message survives.
const stderrLimit = 4096

// NewYAMLLoader builds a loader without profile awareness. When mustExist is
// true, the absence of every candidate file (.yaml and .yml) is reported as an
// error; otherwise the loader silently returns nil.
func NewYAMLLoader(path string, mustExist bool, existingVars ...[]*model.EnvVar) LoadFunc {
	return NewYAMLLoaderWithProfile(path, "", mustExist, existingVars...)
}

// NewYAMLLoaderWithProfile is the profile-aware variant of NewYAMLLoader. See
// NewYAMLLoader for the mustExist semantics.
func NewYAMLLoaderWithProfile(path string, profile string, mustExist bool, existingVars ...[]*model.EnvVar) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)

		// Load both .env.yaml and .env.yml if they exist
		config, err := loadAndMergeYAMLFiles(ctx, path, mustExist)
		if err != nil {
			return nil, err
		}

		if config == nil {
			// No YAML files found
			return nil, nil
		}

		// Merge existing variables if provided
		var allExistingVars []*model.EnvVar
		for _, vars := range existingVars {
			allExistingVars = append(allExistingVars, vars...)
		}

		// Create unified resolver with existing variables
		baseDir := filepath.Dir(path)
		resolver := newYAMLUnifiedResolverWithProfileAndVars(config, profile, baseDir, allExistingVars)

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
				return nil, &model.ConfigFileError{
					Path:   path,
					Format: model.FormatYAML,
					Reason: model.ReasonInvalidSchema,
					Detail: fmt.Sprintf("variable %q: %v", key, err),
					Cause:  err,
				}
			}

			logger.Debug("resolving YAML variable", "key", key)
			resolvedValue, err := resolver.resolveWithValue(key, effectiveValue)
			if err != nil {
				return nil, &model.VariableError{
					Key:     key,
					Path:    path,
					Profile: profile,
					Cause:   err,
				}
			}

			envVar := &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceYAML,
				Secret: value.Secret || effectiveValue.Secret,
			}
			envVars = append(envVars, envVar)
		}

		logger.Debug("loaded YAML file", "path", path, "variables", len(envVars))
		return envVars, nil
	}
}

// loadAndMergeYAMLFiles loads both .env.yaml and .env.yml if they exist and merges them.
// When mustExist is true and neither candidate file exists, a ConfigFileError with
// ReasonNotFound is returned (using the .yaml candidate as the representative path).
func loadAndMergeYAMLFiles(ctx context.Context, path string, mustExist bool) (model.YAMLConfig, error) {
	logger := ctxlog.From(ctx)

	// Helper function to load a single YAML file
	loadOneFile := func(filePath string) (model.YAMLConfig, bool, error) {
		if _, err := os.Stat(filePath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, false, nil // File not found is acceptable
			}
			return nil, false, &model.ConfigFileError{
				Path:   filePath,
				Format: model.FormatYAML,
				Reason: model.ReasonNotReadable,
				Cause:  err,
			}
		}

		logger.Debug("loading YAML file", "path", filePath)
		data, err := os.ReadFile(filePath) // #nosec G304 - file path is user provided and expected
		if err != nil {
			return nil, false, &model.ConfigFileError{
				Path:   filePath,
				Format: model.FormatYAML,
				Reason: model.ReasonNotReadable,
				Cause:  err,
			}
		}

		var config model.YAMLConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, false, &model.ConfigFileError{
				Path:   filePath,
				Format: model.FormatYAML,
				Reason: model.ReasonParseError,
				Detail: err.Error(),
				Cause:  err,
			}
		}

		return config, true, nil
	}

	// Determine base path and construct both .yaml and .yml paths
	base := path
	ext := filepath.Ext(path)
	if ext == ".yaml" || ext == ".yml" {
		base = strings.TrimSuffix(path, ext)
	}
	yamlPath := base + ".yaml"
	ymlPath := base + ".yml"

	// Load .env.yaml
	config1, found1, err1 := loadOneFile(yamlPath)
	if err1 != nil {
		return nil, err1
	}

	// Load .env.yml (only if it's a different file)
	var config2 model.YAMLConfig
	var found2 bool
	if yamlPath != ymlPath {
		var err2 error
		config2, found2, err2 = loadOneFile(ymlPath)
		if err2 != nil {
			return nil, err2
		}
	}

	// If neither file exists, either error (when the caller insisted the file
	// must exist) or stay silent (default-discovery mode).
	if !found1 && !found2 {
		if mustExist {
			return nil, &model.ConfigFileError{
				Path:   yamlPath,
				Format: model.FormatYAML,
				Reason: model.ReasonNotFound,
			}
		}
		logger.Debug("no YAML files found", "yaml_path", yamlPath, "yml_path", ymlPath)
		return nil, nil
	}

	// If only one file exists, return it
	if !found1 {
		logger.Debug("loaded YAML file", "path", ymlPath, "variables", len(config2))
		return config2, nil
	}
	if !found2 || yamlPath == ymlPath {
		logger.Debug("loaded YAML file", "path", yamlPath, "variables", len(config1))
		return config1, nil
	}

	// Both files exist - merge them with conflict detection
	logger.Debug("merging YAML files", "yaml_path", yamlPath, "yml_path", ymlPath)
	merged, err := mergeYAMLConfigs(config1, config2)
	if err != nil {
		return nil, &model.ConfigFileError{
			Path:   yamlPath + " / " + ymlPath,
			Format: model.FormatYAML,
			Reason: model.ReasonInvalidSchema,
			Detail: err.Error(),
			Cause:  err,
		}
	}

	logger.Debug("merged YAML files", "variables", len(merged))
	return merged, nil
}

// mergeYAMLConfigs merges two YAML configurations with field-level conflict detection
func mergeYAMLConfigs(config1, config2 model.YAMLConfig) (model.YAMLConfig, error) {
	result := make(model.YAMLConfig)

	// Copy all entries from config1
	for key, value := range config1 {
		result[key] = value
	}

	// Merge entries from config2
	for key, value2 := range config2 {
		if value1, exists := result[key]; exists {
			// Key exists in both configs - check for field-level conflicts
			merged, err := mergeYAMLValues(key, value1, value2)
			if err != nil {
				return nil, err
			}
			result[key] = merged
		} else {
			// Key only exists in config2
			result[key] = value2
		}
	}

	return result, nil
}

// mergeYAMLValues merges two YAMLValue instances with conflict detection
func mergeYAMLValues(key string, v1, v2 model.YAMLValue) (model.YAMLValue, error) {
	// Check for value source conflicts (value, file, command, alias)
	v1HasValueSource := v1.Value != nil || v1.File != nil || len(v1.Command) > 0 || v1.Alias != nil
	v2HasValueSource := v2.Value != nil || v2.File != nil || len(v2.Command) > 0 || v2.Alias != nil

	if v1HasValueSource && v2HasValueSource {
		// Both have value sources - check if they conflict
		if v1.Value != nil && v2.Value != nil {
			return model.YAMLValue{}, goerr.New(
				fmt.Sprintf("conflicting field \"value\" for environment variable \"%s\" found in both .env.yaml and .env.yml", key),
			)
		}
		if v1.File != nil && v2.File != nil {
			return model.YAMLValue{}, goerr.New(
				fmt.Sprintf("conflicting field \"file\" for environment variable \"%s\" found in both .env.yaml and .env.yml", key),
			)
		}
		if len(v1.Command) > 0 && len(v2.Command) > 0 {
			return model.YAMLValue{}, goerr.New(
				fmt.Sprintf("conflicting field \"command\" for environment variable \"%s\" found in both .env.yaml and .env.yml", key),
			)
		}
		if v1.Alias != nil && v2.Alias != nil {
			return model.YAMLValue{}, goerr.New(
				fmt.Sprintf("conflicting field \"alias\" for environment variable \"%s\" found in both .env.yaml and .env.yml", key),
			)
		}
		// Different value sources - this will be caught by Validate() later
		// We still merge and let validation handle it
	}

	// Merge the values
	merged := model.YAMLValue{}

	// Take value source from whichever has it (only one should have it based on checks above)
	if v1.Value != nil {
		merged.Value = v1.Value
	} else if v2.Value != nil {
		merged.Value = v2.Value
	}

	if v1.File != nil {
		merged.File = v1.File
	} else if v2.File != nil {
		merged.File = v2.File
	}

	if len(v1.Command) > 0 {
		merged.Command = v1.Command
	} else if len(v2.Command) > 0 {
		merged.Command = v2.Command
	}

	if v1.Alias != nil {
		merged.Alias = v1.Alias
	} else if v2.Alias != nil {
		merged.Alias = v2.Alias
	}

	// Merge refs (deduplicate)
	refsMap := make(map[string]bool)
	for _, ref := range v1.Refs {
		refsMap[ref] = true
	}
	for _, ref := range v2.Refs {
		refsMap[ref] = true
	}
	if len(refsMap) > 0 {
		merged.Refs = make([]string, 0, len(refsMap))
		for ref := range refsMap {
			merged.Refs = append(merged.Refs, ref)
		}
	}

	// Merge secret flag (true if either is true)
	merged.Secret = v1.Secret || v2.Secret

	// Merge profiles (v2 overrides v1 for same profile names)
	if len(v1.Profile) > 0 || len(v2.Profile) > 0 {
		merged.Profile = make(map[string]*model.YAMLValue)
		for name, profile := range v1.Profile {
			merged.Profile[name] = profile
		}
		for name, profile := range v2.Profile {
			merged.Profile[name] = profile // v2 overrides v1
		}
	}

	return merged, nil
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
		return "", &model.CommandExecError{Command: command, Cause: goerr.New("command is empty")}
	}
	cmd := exec.Command(command[0], command[1:]...) // #nosec G204 - command is from user-provided YAML config, which is expected
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		exitCode := 0
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		return "", &model.CommandExecError{
			Command:  command,
			ExitCode: exitCode,
			Stderr:   model.TruncateStderr(stderr.String(), stderrLimit),
			Cause:    err,
		}
	}
	return strings.TrimSpace(string(output)), nil
}

// yamlUnifiedResolver handles resolution of all variable types with circular reference detection
type yamlUnifiedResolver struct {
	config       model.YAMLConfig
	profile      string
	baseDir      string // Base directory for resolving relative file paths
	resolvedVars map[string]string
	resolving    map[string]bool   // Track variables currently being resolved
	stack        []string          // Resolution order, used to report circular chains
	externalVars map[string]string // Variables from .env files, system environment, and other sources
}

func newYAMLUnifiedResolverWithProfileAndVars(config model.YAMLConfig, profile string, baseDir string, existingVars []*model.EnvVar) *yamlUnifiedResolver {
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
		baseDir:      baseDir,
		resolvedVars: make(map[string]string),
		resolving:    make(map[string]bool),
		externalVars: externalVars,
	}
}

// buildTemplateContext resolves all refs and builds a context map for template execution.
// Errors from inner resolution are wrapped as ReferenceError so the outer layer
// knows which reference triggered the failure.
func (r *yamlUnifiedResolver) buildTemplateContext(refs []string) (map[string]string, error) {
	context := make(map[string]string)
	for _, ref := range refs {
		refValue, err := r.resolve(ref)
		if err != nil {
			// If resolve already returned a typed ReferenceError (e.g. RefNotFound
			// or RefCircular), propagate it as-is to preserve its semantics.
			var refErr *model.ReferenceError
			if errors.As(err, &refErr) && refErr.Ref == ref {
				return nil, err
			}
			return nil, &model.ReferenceError{
				Ref:    ref,
				Reason: model.RefResolveFailed,
				Cause:  err,
			}
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
		chain := append([]string{}, r.stack...)
		chain = append(chain, key)
		return "", &model.ReferenceError{
			Ref:    key,
			Reason: model.RefCircular,
			Chain:  chain,
		}
	}

	// Mark as currently resolving
	r.resolving[key] = true
	r.stack = append(r.stack, key)
	defer func() {
		delete(r.resolving, key)
		r.stack = r.stack[:len(r.stack)-1]
	}()

	// Get the configuration for this key
	config, exists := r.config[key]
	if !exists {
		// Not in YAML config, check external variables (which includes system vars)
		if value, exists := r.externalVars[key]; exists {
			r.resolvedVars[key] = value
			return value, nil
		}
		// Not found anywhere - return error for missing variable
		return "", &model.ReferenceError{
			Ref:       key,
			Reason:    model.RefNotFound,
			Available: r.availableNames(),
		}
	}

	// Get effective value considering profile
	effectiveValue := config.GetValueForProfile(r.profile)
	return r.resolveWithValue(key, effectiveValue)
}

// availableNames returns a sorted list of names the resolver could resolve.
// Used to populate ReferenceError.Available for "not found" suggestions.
func (r *yamlUnifiedResolver) availableNames() []string {
	names := make([]string, 0, len(r.config)+len(r.externalVars))
	for k := range r.config {
		names = append(names, k)
	}
	for k := range r.externalVars {
		names = append(names, k)
	}
	return names
}

func (r *yamlUnifiedResolver) resolveWithValue(key string, config *model.YAMLValue) (string, error) {
	if config == nil {
		// Indicates a logic bug in the caller; surface it via VariableError so it
		// still goes through the structured formatting path.
		return "", &model.VariableError{
			Key:   key,
			Cause: goerr.New("nil configuration"),
		}
	}

	// Resolve based on type
	var resolvedValue string
	var err error

	switch {
	case config.Value != nil:
		// If refs are present, treat value as a template
		if len(config.Refs) > 0 {
			// Build context for template; ReferenceError-typed errors are passed
			// through so the formatter can describe which ref failed.
			context, err := r.buildTemplateContext(config.Refs)
			if err != nil {
				return "", err
			}

			// Parse and execute template
			tmpl, err := template.New("env").Parse(*config.Value)
			if err != nil {
				return "", &model.ResolveError{
					Op:     model.OpTemplate,
					Target: *config.Value,
					Cause:  err,
				}
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, context); err != nil {
				return "", &model.ResolveError{
					Op:     model.OpTemplate,
					Target: *config.Value,
					Cause:  err,
				}
			}

			resolvedValue = buf.String()
		} else {
			// No refs, use value as-is
			resolvedValue = *config.Value
		}

	case config.File != nil:
		filePath := *config.File
		if !filepath.IsAbs(filePath) && r.baseDir != "" {
			filePath = filepath.Join(r.baseDir, filePath)
		}
		resolvedValue, err = readYAMLFile(filePath)
		if err != nil {
			return "", &model.ResolveError{
				Op:     model.OpReadFile,
				Target: *config.File,
				Cause:  err,
			}
		}

	case len(config.Command) > 0:
		// Resolve command with optional refs
		commandToExecute := config.Command

		// If refs are present, resolve them and apply templates to command elements
		if len(config.Refs) > 0 {
			// Build context for template
			context, err := r.buildTemplateContext(config.Refs)
			if err != nil {
				return "", err
			}

			// Apply template to each command element
			resolvedCommand := make([]string, len(config.Command))
			tmpl := template.New("cmd")
			for i, cmdElement := range config.Command {
				parsedTmpl, err := tmpl.Parse(cmdElement)
				if err != nil {
					return "", &model.ResolveError{
						Op:     model.OpTemplate,
						Target: cmdElement,
						Cause:  err,
					}
				}

				var buf bytes.Buffer
				if err := parsedTmpl.Execute(&buf, context); err != nil {
					return "", &model.ResolveError{
						Op:     model.OpTemplate,
						Target: cmdElement,
						Cause:  err,
					}
				}
				resolvedCommand[i] = buf.String()
			}
			commandToExecute = resolvedCommand
		}

		// executeYAMLCommand already returns a CommandExecError on failure.
		resolvedValue, err = executeYAMLCommand(commandToExecute)
		if err != nil {
			return "", err
		}

	case config.Alias != nil:
		// Recursively resolve the alias target.
		resolvedValue, err = r.resolve(*config.Alias)
		if err != nil {
			// If resolve already returned ReferenceError-typed, pass through.
			var refErr *model.ReferenceError
			if errors.As(err, &refErr) && refErr.Ref == *config.Alias {
				return "", err
			}
			return "", &model.ReferenceError{
				Ref:    *config.Alias,
				Reason: model.RefResolveFailed,
				Cause:  err,
			}
		}
	}

	// Cache the resolved value
	r.resolvedVars[key] = resolvedValue
	return resolvedValue, nil
}
