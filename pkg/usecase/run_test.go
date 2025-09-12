package usecase_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"github.com/m-mizutani/zenv/v2/pkg/usecase"
)

func TestUseCase(t *testing.T) {

	t.Run("Run with inline environment variables only", func(t *testing.T) {
		var executedCmd string
		var executedArgs []string
		var executedEnvVars []*model.EnvVar

		mockExecutor := func(cmd string, args []string, envVars []*model.EnvVar) (int, error) {
			executedCmd = cmd
			executedArgs = args
			executedEnvVars = envVars
			return 0, nil
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

		mockExecutor := func(cmd string, args []string, envVars []*model.EnvVar) (int, error) {
			executedEnvVars = envVars
			return 0, nil
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

		mockExecutor := func(cmd string, args []string, envVars []*model.EnvVar) (int, error) {
			executedEnvVars = envVars
			return 0, nil
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
				{Name: "LOADER2_VAR", Value: "loader2_value", Source: model.SourceTOML},
			}, nil
		}

		mockExecutor := func(cmd string, args []string, envVars []*model.EnvVar) (int, error) {
			executedEnvVars = envVars
			return 0, nil
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
		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		mockLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return []*model.EnvVar{
				{Name: "TEST_VAR", Value: "test_value", Source: model.SourceDotEnv},
			}, nil
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{mockLoader}, executor.NewDefaultExecutor())

		err := uc.Run(context.Background(), []string{})

		// Restore stdout and read all captured output
		w.Close()
		os.Stdout = oldStdout
		output := string(gt.R1(io.ReadAll(r)).NoError(t))

		gt.NoError(t, err)
		gt.S(t, output).Contains("TEST_VAR=test_value")
		gt.S(t, output).Contains("[.env]")
	})

	t.Run("Show environment variables with inline vars only", func(t *testing.T) {
		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		uc := usecase.NewUseCase([]loader.LoadFunc{}, executor.NewDefaultExecutor())

		// Call with inline var but no command (empty args after inline var)
		err := uc.Run(context.Background(), []string{"INLINE_VAR=inline_value"})

		// Restore stdout and read all captured output
		w.Close()
		os.Stdout = oldStdout
		output := string(gt.R1(io.ReadAll(r)).NoError(t))

		gt.NoError(t, err)
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

		mockExecutor := func(cmd string, args []string, envVars []*model.EnvVar) (int, error) {
			executedEnvVars = envVars
			return 0, nil
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
}
