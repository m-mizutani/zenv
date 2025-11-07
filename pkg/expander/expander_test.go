package expander_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/expander"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestExpander_Expand(t *testing.T) {
	ctx := context.Background()

	envVars := []*model.EnvVar{
		{Name: "NAME", Value: "world"},
		{Name: "GREETING", Value: "Hello"},
		{Name: "COUNT", Value: "42"},
	}

	exp := expander.NewExpander(envVars)

	tests := []struct {
		name     string
		template string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple variable expansion",
			template: "{{ .NAME }}",
			expected: "world",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "{{ .GREETING }}, {{ .NAME }}!",
			expected: "Hello, world!",
			wantErr:  false,
		},
		{
			name:     "template with text",
			template: "The answer is {{ .COUNT }}",
			expected: "The answer is 42",
			wantErr:  false,
		},
		{
			name:     "no template",
			template: "plain text",
			expected: "plain text",
			wantErr:  false,
		},
		{
			name:     "empty string",
			template: "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "conditional template",
			template: `{{ if eq .COUNT "42" }}correct{{ else }}wrong{{ end }}`,
			expected: "correct",
			wantErr:  false,
		},
		{
			name:     "undefined variable",
			template: "{{ .UNDEFINED }}",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "invalid template syntax",
			template: "{{ .NAME",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "extra braces are treated as text",
			template: "{{ .NAME }}}}",
			expected: "world}}",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exp.Expand(ctx, tt.template)
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.Equal(t, result, tt.expected)
			}
		})
	}
}

func TestExpander_ExpandArgs(t *testing.T) {
	ctx := context.Background()

	envVars := []*model.EnvVar{
		{Name: "HOST", Value: "localhost"},
		{Name: "PORT", Value: "8080"},
		{Name: "USER", Value: "admin"},
	}

	exp := expander.NewExpander(envVars)

	tests := []struct {
		name     string
		args     []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "expand multiple arguments",
			args:     []string{"--host", "{{ .HOST }}", "--port", "{{ .PORT }}"},
			expected: []string{"--host", "localhost", "--port", "8080"},
			wantErr:  false,
		},
		{
			name:     "mixed template and plain text",
			args:     []string{"connect", "{{ .USER }}@{{ .HOST }}:{{ .PORT }}"},
			expected: []string{"connect", "admin@localhost:8080"},
			wantErr:  false,
		},
		{
			name:     "no templates",
			args:     []string{"command", "arg1", "arg2"},
			expected: []string{"command", "arg1", "arg2"},
			wantErr:  false,
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "error in one argument",
			args:     []string{"valid", "{{ .UNDEFINED }}"},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := exp.ExpandArgs(ctx, tt.args)
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
				gt.Equal(t, result, tt.expected)
			}
		})
	}
}

func TestNewExpander(t *testing.T) {
	envVars := []*model.EnvVar{
		{Name: "VAR1", Value: "value1"},
		{Name: "VAR2", Value: "value2"},
	}

	exp := expander.NewExpander(envVars)
	gt.NotNil(t, exp)

	// Test that the expander was initialized correctly
	ctx := context.Background()
	result, err := exp.Expand(ctx, "{{ .VAR1 }}")
	gt.NoError(t, err)
	gt.Equal(t, result, "value1")
}

func TestExpander_EmptyEnvVars(t *testing.T) {
	ctx := context.Background()
	exp := expander.NewExpander([]*model.EnvVar{})

	// Should work with no variables
	result, err := exp.Expand(ctx, "plain text")
	gt.NoError(t, err)
	gt.Equal(t, result, "plain text")

	// Should fail when trying to access undefined variable
	_, err = exp.Expand(ctx, "{{ .UNDEFINED }}")
	gt.Error(t, err)
}
