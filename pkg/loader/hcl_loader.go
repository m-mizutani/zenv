package loader

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"github.com/zclconf/go-cty/cty"
)

// NewHCLLoader creates a loader for HCL configuration files. When mustExist is
// true, a missing file is reported as an error; otherwise it is silently
// skipped (used for the default .env.hcl path).
func NewHCLLoader(path string, mustExist bool, existingVars ...[]*model.EnvVar) LoadFunc {
	return NewHCLLoaderWithProfile(path, "", mustExist, existingVars...)
}

// NewHCLLoaderWithProfile creates a profile-aware loader for HCL configuration files.
// See NewHCLLoader for the mustExist semantics.
func NewHCLLoaderWithProfile(path string, profile string, mustExist bool, existingVars ...[]*model.EnvVar) LoadFunc {
	return func(ctx context.Context) ([]*model.EnvVar, error) {
		logger := ctxlog.From(ctx)

		config, err := loadHCLFile(ctx, path, mustExist)
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
		resolver := newYAMLUnifiedResolverWithProfileAndVars(ctx, config, profile, baseDir, allExistingVars)

		var envVars []*model.EnvVar
		for key, value := range config {
			effectiveValue := value.GetValueForProfile(profile)

			if effectiveValue == nil || effectiveValue.IsEmpty() {
				logger.Debug("skipping variable (unset or not defined in profile)", "key", key, "profile", profile)
				continue
			}

			if err := effectiveValue.Validate(); err != nil {
				return nil, &model.ConfigFileError{
					Path:   path,
					Format: model.FormatHCL,
					Reason: model.ReasonInvalidSchema,
					Detail: fmt.Sprintf("variable %q: %v", key, err),
					Cause:  err,
				}
			}

			logger.Debug("resolving HCL variable", "key", key)
			resolvedValue, err := resolver.resolveWithValue(key, effectiveValue)
			if err != nil {
				return nil, &model.VariableError{
					Key:     key,
					Path:    path,
					Profile: profile,
					Cause:   err,
				}
			}

			envVars = append(envVars, &model.EnvVar{
				Name:   key,
				Value:  resolvedValue,
				Source: model.SourceHCL,
				Secret: value.Secret || effectiveValue.Secret,
			})
		}

		logger.Debug("loaded HCL file", "path", path, "variables", len(envVars))
		return envVars, nil
	}
}

func loadHCLFile(ctx context.Context, path string, mustExist bool) (model.YAMLConfig, error) {
	logger := ctxlog.From(ctx)

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if mustExist {
				return nil, &model.ConfigFileError{
					Path:   path,
					Format: model.FormatHCL,
					Reason: model.ReasonNotFound,
					Cause:  err,
				}
			}
			return nil, nil
		}
		return nil, &model.ConfigFileError{
			Path:   path,
			Format: model.FormatHCL,
			Reason: model.ReasonNotReadable,
			Cause:  err,
		}
	}

	logger.Debug("loading HCL file", "path", path)
	data, err := os.ReadFile(path) // #nosec G304 - file path is user provided and expected
	if err != nil {
		return nil, &model.ConfigFileError{
			Path:   path,
			Format: model.FormatHCL,
			Reason: model.ReasonNotReadable,
			Cause:  err,
		}
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, path)
	if diags.HasErrors() {
		return nil, &model.ConfigFileError{
			Path:   path,
			Format: model.FormatHCL,
			Reason: model.ReasonParseError,
			// diags.Error() collapses multiple diagnostics to "first, and N
			// other(s)" and never shows the offending source line, so render
			// each diagnostic with its location, a source snippet and a caret
			// instead — that is what makes a parse failure actionable.
			Detail: renderHCLDiagnostics(diags, data),
		}
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, &model.ConfigFileError{
			Path:   path,
			Format: model.FormatHCL,
			Reason: model.ReasonInvalidSchema,
			Detail: "unexpected HCL body type",
		}
	}

	cfg, err := parseHCLBody(body)
	if err != nil {
		// parseHCLBody returns goerr-based schema errors; wrap them as
		// ConfigFileError so the formatter can attach the file path.
		// Pass through if already typed (even when wrapped).
		var cfe *model.ConfigFileError
		if errors.As(err, &cfe) {
			return nil, err
		}
		return nil, &model.ConfigFileError{
			Path:   path,
			Format: model.FormatHCL,
			Reason: model.ReasonInvalidSchema,
			Detail: err.Error(),
			Cause:  err,
		}
	}
	return cfg, nil
}

// renderHCLDiagnostics turns parse diagnostics into a multi-line, human-readable
// report. Each diagnostic shows its location, the offending source line and a
// caret pointing at the column, followed by HCL's own remediation prose. We hand-
// roll this instead of using hcl.NewDiagnosticTextWriter so the output nests
// cleanly under the CLI error formatter (no duplicated "Error:" headers or file
// paths, both of which the formatter already prints).
func renderHCLDiagnostics(diags hcl.Diagnostics, src []byte) string {
	srcLines := strings.Split(string(src), "\n")

	var b strings.Builder
	for i, d := range diags {
		// Separate diagnostics with a blank line so a multi-diagnostic report
		// does not read as one dense block.
		if i > 0 {
			b.WriteString("\n\n")
		}

		if d.Subject != nil {
			fmt.Fprintf(&b, "line %d, column %d: %s", d.Subject.Start.Line, d.Subject.Start.Column, d.Summary)
		} else {
			b.WriteString(d.Summary)
		}

		if snippet := hclSnippet(srcLines, d.Subject); snippet != "" {
			b.WriteString("\n")
			b.WriteString(snippet)
		}

		// HCL's remediation prose is usually a single line, but indent every
		// line so a multi-line detail stays aligned under its diagnostic.
		if d.Detail != "" {
			for line := range strings.SplitSeq(d.Detail, "\n") {
				b.WriteString("\n    ")
				b.WriteString(line)
			}
		}
	}
	return b.String()
}

// hclSnippet renders the source line referenced by rng plus a caret line
// underneath it. Tabs are expanded to a single space so the caret column stays
// aligned with the rendered text. Returns "" when the range is missing or
// points outside the available source.
func hclSnippet(srcLines []string, rng *hcl.Range) string {
	if rng == nil || rng.Start.Line < 1 || rng.Start.Line > len(srcLines) {
		return ""
	}
	// Strip a trailing CR so CRLF-terminated source lines do not emit a stray
	// carriage return that scrambles the rendered snippet, then expand tabs to
	// a single space so the caret column stays aligned with the text.
	line := strings.TrimSuffix(srcLines[rng.Start.Line-1], "\r")
	line = strings.ReplaceAll(line, "\t", " ")

	caretCol := max(rng.Start.Column, 1)
	caretLen := 1
	// A single-line range tells us how many columns the offending token spans.
	if rng.End.Line == rng.Start.Line && rng.End.Column > rng.Start.Column {
		caretLen = rng.End.Column - rng.Start.Column
	}

	var b strings.Builder
	b.WriteString("    ")
	b.WriteString(line)
	b.WriteString("\n    ")
	b.WriteString(strings.Repeat(" ", caretCol-1))
	b.WriteString(strings.Repeat("^", caretLen))
	return b.String()
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
			return nil, goerr.New("duplicate variable name",
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
		case "aws_secret":
			s, err := evalStringAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid aws_secret attribute")
			}
			v.AWSSecret = s
		case "gcp_secret":
			s, err := evalStringAttr(attr)
			if err != nil {
				return v, goerr.Wrap(err, "invalid gcp_secret attribute")
			}
			v.GCPSecret = s
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
