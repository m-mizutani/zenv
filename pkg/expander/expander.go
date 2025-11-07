package expander

import (
	"bytes"
	"context"
	"text/template"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// Expander provides template expansion functionality for command arguments
type Expander struct {
	envVars map[string]string
}

// NewExpander creates a new template expander with environment variables
func NewExpander(envVars []*model.EnvVar) *Expander {
	envMap := make(map[string]string, len(envVars))
	for _, ev := range envVars {
		envMap[ev.Name] = ev.Value
	}
	return &Expander{
		envVars: envMap,
	}
}

// Expand expands a single template string using the environment variables
func (e *Expander) Expand(ctx context.Context, tmpl string) (string, error) {
	logger := ctxlog.From(ctx)
	logger.Debug("expanding template", "template", tmpl)

	t, err := template.New("").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, e.envVars); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}

	result := buf.String()
	logger.Debug("template expanded", "template", tmpl, "result", result)
	return result, nil
}

// ExpandArgs expands multiple template strings (command arguments)
func (e *Expander) ExpandArgs(ctx context.Context, args []string) ([]string, error) {
	logger := ctxlog.From(ctx)
	logger.Debug("expanding template args", "args", args)

	expanded := make([]string, len(args))
	for i, arg := range args {
		result, err := e.Expand(ctx, arg)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to expand argument", goerr.V("index", i), goerr.V("arg", arg))
		}
		expanded[i] = result
	}

	logger.Debug("template args expanded", "expanded", expanded)
	return expanded, nil
}
