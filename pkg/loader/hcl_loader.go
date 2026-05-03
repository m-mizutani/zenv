package loader

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"github.com/zclconf/go-cty/cty"
)

// NewHCLLoader creates a loader for HCL configuration files.
func NewHCLLoader(path string, existingVars ...[]*model.EnvVar) LoadFunc {
	return NewHCLLoaderWithProfile(path, "", existingVars...)
}

// NewHCLLoaderWithProfile creates a profile-aware loader for HCL configuration files.
func NewHCLLoaderWithProfile(path string, profile string, existingVars ...[]*model.EnvVar) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)

		config, err := loadHCLFile(ctx, path)
		if err != nil {
			return nil, err
		}
		if config == nil {
			return nil, nil
		}

		var allExistingVars []*model.EnvVar
		for _, vars := range existingVars {
			allExistingVars = append(allExistingVars, vars...)
		}

		// Reuse the YAML resolver since the in-memory representation is identical.
		baseDir := filepath.Dir(path)
		resolver := newYAMLUnifiedResolverWithProfileAndVars(config, profile, baseDir, allExistingVars)

		var envVars []*model.EnvVar
		for key, value := range config {
			effectiveValue := value.GetValueForProfile(profile)

			if effectiveValue == nil || effectiveValue.IsEmpty() {
				logger.Debug("skipping variable (unset or not defined in profile)", "key", key, "profile", profile)
				continue
			}

			if err := effectiveValue.Validate(); err != nil {
				logger.Error("invalid HCL configuration", "key", key, "error", err)
				return nil, goerr.Wrap(err, "invalid configuration", goerr.V("key", key))
			}

			logger.Debug("resolving HCL variable", "key", key)
			resolvedValue, err := resolver.resolveWithValue(key, effectiveValue)
			if err != nil {
				logger.Error("failed to resolve HCL variable", "key", key, "error", err)
				return nil, goerr.Wrap(err, "failed to resolve variable", goerr.V("key", key))
			}

			envVars = append(envVars, &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceYAML,
				Secret: value.Secret || effectiveValue.Secret,
			})
		}

		logger.Debug("loaded HCL file", "path", path, "variables", len(envVars))
		return envVars, nil
	}
}

func loadHCLFile(ctx context.Context, path string) (model.YAMLConfig, error) {
	logger := ctxlog.From(ctx)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to check HCL file", goerr.V("path", path))
	}

	logger.Debug("loading HCL file", "path", path)
	data, err := os.ReadFile(path) // #nosec G304 - file path is user provided and expected
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read HCL file", goerr.V("path", path))
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, path)
	if diags.HasErrors() {
		return nil, goerr.New("failed to parse HCL file",
			goerr.V("path", path),
			goerr.V("diagnostics", diags.Error()))
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, goerr.New("unexpected HCL body type", goerr.V("path", path))
	}

	return parseHCLBody(body)
}

// parseHCLBody converts the top-level body of an HCL file into a YAMLConfig.
// Attributes (KEY = "value") become scalar variables, and blocks (KEY { ... })
// become structured variables.
func parseHCLBody(body *hclsyntax.Body) (model.YAMLConfig, error) {
	config := make(model.YAMLConfig)

	for name, attr := range body.Attributes {
		s, err := evalStringAttr(attr)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read attribute", goerr.V("name", name))
		}
		// Top-level attribute: capture as scalar value (allow null = no entry).
		if s == nil {
			continue
		}
		config[name] = model.YAMLValue{Value: s}
	}

	for _, block := range body.Blocks {
		name := block.Type
		if _, exists := config[name]; exists {
			return nil, goerr.New("duplicate variable name (defined as both attribute and block)",
				goerr.V("name", name))
		}
		if len(block.Labels) > 0 {
			return nil, goerr.New("block labels are not supported",
				goerr.V("name", name),
				goerr.V("labels", block.Labels))
		}

		v, err := parseValueBlock(block.Body)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse block", goerr.V("name", name))
		}
		config[name] = v
	}

	return config, nil
}

// parseValueBlock parses a block body that represents a single environment variable
// definition (value/file/command/alias/refs/secret/profile).
func parseValueBlock(body *hclsyntax.Body) (model.YAMLValue, error) {
	var v model.YAMLValue

	for name, attr := range body.Attributes {
		switch name {
		case "value":
			s, err := evalStringAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid value attribute")
			}
			v.Value = s
		case "file":
			s, err := evalStringAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid file attribute")
			}
			v.File = s
		case "alias":
			s, err := evalStringAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid alias attribute")
			}
			v.Alias = s
		case "command":
			arr, err := evalStringSliceAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid command attribute")
			}
			v.Command = arr
		case "refs":
			arr, err := evalStringSliceAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid refs attribute")
			}
			v.Refs = arr
		case "secret":
			b, err := evalBoolAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid secret attribute")
			}
			v.Secret = b
		default:
			return v, goerr.New("unknown attribute in value block", goerr.V("name", name))
		}
	}

	for _, block := range body.Blocks {
		switch block.Type {
		case "profile":
			if v.Profile != nil {
				return v, goerr.New("multiple profile blocks are not allowed")
			}
			profile, err := parseProfileBlock(block.Body)
			if err != nil {
				return v, goerr.Wrap(err, "failed to parse profile block")
			}
			v.Profile = profile
		default:
			return v, goerr.New("unknown nested block type", goerr.V("type", block.Type))
		}
	}

	return v, nil
}

// parseProfileBlock parses a profile { ... } block body. Each entry can be either:
//   - attribute (dev = "value"): treated as a scalar value
//   - attribute = null: treated as an explicit unset (empty YAMLValue)
//   - block (dev { value = ..., file = ... }): treated as a structured value
func parseProfileBlock(body *hclsyntax.Body) (map[string]*model.YAMLValue, error) {
	profile := make(map[string]*model.YAMLValue)

	for name, attr := range body.Attributes {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, goerr.New("failed to evaluate profile attribute",
				goerr.V("name", name),
				goerr.V("diagnostics", diags.Error()))
		}

		if val.IsNull() {
			// null marks the profile as unset
			profile[name] = &model.YAMLValue{}
			continue
		}

		if val.Type() != cty.String {
			return nil, goerr.New("profile attribute must be string or null",
				goerr.V("name", name),
				goerr.V("got", val.Type().FriendlyName()))
		}

		s := val.AsString()
		profile[name] = &model.YAMLValue{Value: &s}
	}

	for _, block := range body.Blocks {
		name := block.Type
		if _, exists := profile[name]; exists {
			return nil, goerr.New("duplicate profile entry", goerr.V("name", name))
		}
		if len(block.Labels) > 0 {
			return nil, goerr.New("profile entry labels are not supported",
				goerr.V("name", name),
				goerr.V("labels", block.Labels))
		}

		v, err := parseValueBlock(block.Body)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse profile entry", goerr.V("name", name))
		}
		profile[name] = &v
	}

	return profile, nil
}

func evalStringAttr(attr *hclsyntax.Attribute) (*string, error) {
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return nil, goerr.New("failed to evaluate", goerr.V("diagnostics", diags.Error()))
	}
	if val.IsNull() {
		return nil, nil
	}
	if val.Type() != cty.String {
		return nil, goerr.New("expected string", goerr.V("got", val.Type().FriendlyName()))
	}
	s := val.AsString()
	return &s, nil
}

func evalStringSliceAttr(attr *hclsyntax.Attribute) ([]string, error) {
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return nil, goerr.New("failed to evaluate", goerr.V("diagnostics", diags.Error()))
	}
	if val.IsNull() {
		return nil, nil
	}
	t := val.Type()
	if !t.IsTupleType() && !t.IsListType() {
		return nil, goerr.New("expected list of strings", goerr.V("got", t.FriendlyName()))
	}

	var result []string
	it := val.ElementIterator()
	for it.Next() {
		_, elem := it.Element()
		if elem.IsNull() {
			return nil, goerr.New("list element must not be null")
		}
		if elem.Type() != cty.String {
			return nil, goerr.New("list element must be string", goerr.V("got", elem.Type().FriendlyName()))
		}
		result = append(result, elem.AsString())
	}
	return result, nil
}

func evalBoolAttr(attr *hclsyntax.Attribute) (bool, error) {
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() {
		return false, goerr.New("failed to evaluate", goerr.V("diagnostics", diags.Error()))
	}
	if val.IsNull() {
		return false, nil
	}
	if val.Type() != cty.Bool {
		return false, goerr.New("expected bool", goerr.V("got", val.Type().FriendlyName()))
	}
	return val.True(), nil
}
