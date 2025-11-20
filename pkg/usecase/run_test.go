package usecase_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"github.com/m-mizutani/zenv/v2/pkg/usecase"
)

// testShowEnvVarsOutput is a helper function to capture stdout when showing environment variables
func testShowEnvVarsOutput(t *testing.T, envs []*model.EnvVar) string {
	t.Helper()

	r, w, err := os.Pipe()
	gt.NoError(t, err)

	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
		return envs, nil
	}
	uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, executor.NewDefaultExecutor())

	runErr := uc.Run(context.Background(), []string{})
	gt.NoError(t, runErr)

	w.Close()
	output, readErr := io.ReadAll(r)
	gt.NoError(t, readErr)

	return string(output)
}

func TestUseCase(t *testing.T) {

	t.Run("Run with inline environment variables only", func(t *testing.T) {
		var executedCmd string
		var executedArgs []string
		var executedEnvVars []*model.EnvVar

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedCmd = cmd
			executedArgs = args
			executedEnvVars = envVars
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{}, mockExecutor)

		err := uc.Run(context.Background(), []string{"VAR1=value1", "VAR2=value2", "echo", "hello"})

		gt.NoError(t, err)
		gt.Equal(t, executedCmd, "echo")
		gt.Equal(t, len(executedArgs), 1)
		gt.Equal(t, executedArgs[0], "hello")

		// Check that inline variables are present
		inlineVars := make(map[string]string)
		for _, envVar := range executedEnvVars {
			if envVar.Source == model.SourceInline {
				inlineVars[envVar.Name] = envVar.Value
			}
		}
		gt.Equal(t, inlineVars["VAR1"], "value1")
		gt.Equal(t, inlineVars["VAR2"], "value2")
	})

	t.Run("Run with loader environment variables", func(t *testing.T) {
		var executedEnvVars []*model.EnvVar

		mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "LOADER_VAR", Value: "loader_value", Source: model.SourceDotEnv},
			}, nil
		}

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedEnvVars = envVars
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, mockExecutor)

		err := uc.Run(context.Background(), []string{"echo", "test"})

		gt.NoError(t, err)

		// Check that loader variable is present
		found := false
		for _, envVar := range executedEnvVars {
			if envVar.Name == "LOADER_VAR" && envVar.Value == "loader_value" && envVar.Source == model.SourceDotEnv {
				found = true
				break
			}
		}
		gt.True(t, found)
	})

	t.Run("Variable precedence: inline overrides loader", func(t *testing.T) {
		var executedEnvVars []*model.EnvVar

		mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "CONFLICT_VAR", Value: "loader_value", Source: model.SourceDotEnv},
			}, nil
		}

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedEnvVars = envVars
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, mockExecutor)

		err := uc.Run(context.Background(), []string{"CONFLICT_VAR=inline_value", "echo", "test"})

		gt.NoError(t, err)

		// Check that inline variable overrides loader variable
		for _, envVar := range executedEnvVars {
			if envVar.Name == "CONFLICT_VAR" {
				gt.Equal(t, envVar.Value, "inline_value")
				gt.Equal(t, envVar.Source, model.SourceInline)
				return
			}
		}
		t.Error("CONFLICT_VAR not found in environment variables")
	})

	t.Run("Multiple loaders", func(t *testing.T) {
		var executedEnvVars []*model.EnvVar

		loader1 := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "LOADER1_VAR", Value: "loader1_value", Source: model.SourceDotEnv},
			}, nil
		}

		loader2 := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "LOADER2_VAR", Value: "loader2_value", Source: model.SourceYAML},
			}, nil
		}

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedEnvVars = envVars
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{loader1, loader2}, mockExecutor)

		err := uc.Run(context.Background(), []string{"echo", "test"})

		gt.NoError(t, err)

		// Check that both loader variables are present
		foundLoader1 := false
		foundLoader2 := false
		for _, envVar := range executedEnvVars {
			if envVar.Name == "LOADER1_VAR" && envVar.Value == "loader1_value" {
				foundLoader1 = true
			}
			if envVar.Name == "LOADER2_VAR" && envVar.Value == "loader2_value" {
				foundLoader2 = true
			}
		}
		gt.True(t, foundLoader1)
		gt.True(t, foundLoader2)
	})

	t.Run("Show environment variables when no command specified", func(t *testing.T) {
		output := testShowEnvVarsOutput(t, []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceDotEnv},
		})

		gt.S(t, output).Contains("TEST_VAR=test_value")
		gt.S(t, output).Contains("[.env]")
	})

	t.Run("Show environment variables with inline vars only", func(t *testing.T) {
		output := testShowEnvVarsOutput(t, []*model.EnvVar{
			{Name: "INLINE_VAR", Value: "inline_value", Source: model.SourceInline},
		})

		gt.S(t, output).Contains("INLINE_VAR=inline_value")
		gt.S(t, output).Contains("[inline]")
	})

	t.Run("Handle loader error", func(t *testing.T) {
		errorLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return nil, os.ErrNotExist
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{errorLoader}, executor.NewDefaultExecutor())

		err := uc.Run(context.Background(), []string{"echo", "test"})

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to load environment variables")
	})

	t.Run("Parse inline environment variables correctly", func(t *testing.T) {
		var executedEnvVars []*model.EnvVar

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedEnvVars = envVars
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{}, mockExecutor)

		// Test parsing complex inline variables
		err := uc.Run(context.Background(), []string{"VAR_WITH_EQUALS=value=with=equals", "EMPTY_VAR=", "echo", "test"})

		gt.NoError(t, err)

		// Check inline variables
		inlineVars := make(map[string]string)
		for _, envVar := range executedEnvVars {
			if envVar.Source == model.SourceInline {
				inlineVars[envVar.Name] = envVar.Value
			}
		}

		gt.Equal(t, inlineVars["VAR_WITH_EQUALS"], "value=with=equals")
		gt.Equal(t, inlineVars["EMPTY_VAR"], "")
	})

	t.Run("Template expansion enabled", func(t *testing.T) {
		var executedCmd string
		var executedArgs []string

		mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "HOST", Value: "localhost", Source: model.SourceDotEnv},
				{Name: "PORT", Value: "8080", Source: model.SourceDotEnv},
			}, nil
		}

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedCmd = cmd
			executedArgs = args
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, mockExecutor)
		uc.EnableTemplate = true

		err := uc.Run(context.Background(), []string{"curl", "http://{{ .HOST }}:{{ .PORT }}/api"})

		gt.NoError(t, err)
		gt.Equal(t, executedCmd, "curl")
		gt.Equal(t, len(executedArgs), 1)
		gt.Equal(t, executedArgs[0], "http://localhost:8080/api")
	})

	t.Run("Template expansion with inline variables", func(t *testing.T) {
		var executedArgs []string

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedArgs = args
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{}, mockExecutor)
		uc.EnableTemplate = true

		err := uc.Run(context.Background(), []string{"NAME=world", "GREETING=Hello", "echo", "{{ .GREETING }}, {{ .NAME }}!"})

		gt.NoError(t, err)
		gt.Equal(t, len(executedArgs), 1)
		gt.Equal(t, executedArgs[0], "Hello, world!")
	})

	t.Run("Template expansion disabled (default)", func(t *testing.T) {
		var executedArgs []string

		mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "VAR", Value: "value", Source: model.SourceDotEnv},
			}, nil
		}

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedArgs = args
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, mockExecutor)
		// EnableTemplate is false by default

		err := uc.Run(context.Background(), []string{"echo", "{{ .VAR }}"})

		gt.NoError(t, err)
		gt.Equal(t, len(executedArgs), 1)
		gt.Equal(t, executedArgs[0], "{{ .VAR }}")
	})

	t.Run("Template expansion error: undefined variable", func(t *testing.T) {
		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{}, mockExecutor)
		uc.EnableTemplate = true

		err := uc.Run(context.Background(), []string{"echo", "{{ .UNDEFINED_VAR }}"})

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to expand template arguments")
	})

	t.Run("Template expansion error: invalid syntax", func(t *testing.T) {
		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{}, mockExecutor)
		uc.EnableTemplate = true

		err := uc.Run(context.Background(), []string{"echo", "{{ .VAR"})

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to expand template arguments")
	})

	t.Run("Template expansion with multiple arguments", func(t *testing.T) {
		var executedArgs []string

		mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "USER", Value: "admin", Source: model.SourceDotEnv},
				{Name: "HOST", Value: "localhost", Source: model.SourceDotEnv},
				{Name: "PORT", Value: "5432", Source: model.SourceDotEnv},
			}, nil
		}

		mockExecutor := func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
			executedArgs = args
			return nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, mockExecutor)
		uc.EnableTemplate = true

		err := uc.Run(context.Background(), []string{"psql", "-U", "{{ .USER }}", "-h", "{{ .HOST }}", "-p", "{{ .PORT }}"})

		gt.NoError(t, err)
		gt.Equal(t, len(executedArgs), 6)
		gt.Equal(t, executedArgs[0], "-U")
		gt.Equal(t, executedArgs[1], "admin")
		gt.Equal(t, executedArgs[2], "-h")
		gt.Equal(t, executedArgs[3], "localhost")
		gt.Equal(t, executedArgs[4], "-p")
		gt.Equal(t, executedArgs[5], "5432")
	})

	t.Run("Environment variables are sorted alphabetically when displayed", func(t *testing.T) {
		output := testShowEnvVarsOutput(t, []*model.EnvVar{
			{Name: "ZEBRA", Value: "last", Source: model.SourceDotEnv},
			{Name: "APPLE", Value: "first", Source: model.SourceDotEnv},
			{Name: "MIDDLE", Value: "mid", Source: model.SourceDotEnv},
			{Name: "banana", Value: "lowercase", Source: model.SourceInline},
		})

		// Check that variables appear in alphabetical order (case-insensitive)
		appleIdx := strings.Index(output, "APPLE=first")
		bananaIdx := strings.Index(output, "banana=lowercase")
		middleIdx := strings.Index(output, "MIDDLE=mid")
		zebraIdx := strings.Index(output, "ZEBRA=last")

		gt.True(t, appleIdx < bananaIdx)
		gt.True(t, bananaIdx < middleIdx)
		gt.True(t, middleIdx < zebraIdx)
	})

	t.Run("Environment variables sorting is case-insensitive", func(t *testing.T) {
		output := testShowEnvVarsOutput(t, []*model.EnvVar{
			{Name: "aaa", Value: "1", Source: model.SourceDotEnv},
			{Name: "AAB", Value: "2", Source: model.SourceDotEnv},
			{Name: "Aac", Value: "3", Source: model.SourceDotEnv},
		})

		// Check order: aaa < AAB < Aac
		aaaIdx := strings.Index(output, "aaa=1")
		aabIdx := strings.Index(output, "AAB=2")
		aacIdx := strings.Index(output, "Aac=3")

		gt.True(t, aaaIdx < aabIdx)
		gt.True(t, aabIdx < aacIdx)
	})
}
