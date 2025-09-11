package usecase_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/zenv/pkg/executor"
	"github.com/m-mizutani/zenv/pkg/loader"
	"github.com/m-mizutani/zenv/pkg/model"
	"github.com/m-mizutani/zenv/pkg/usecase"
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

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if executedCmd != "echo" {
			t.Errorf("expected command 'echo', got '%s'", executedCmd)
		}
		if len(executedArgs) != 1 || executedArgs[0] != "hello" {
			t.Errorf("expected args ['hello'], got %v", executedArgs)
		}

		// Check that inline variables are present
		inlineVars := make(map[string]string)
		for _, envVar := range executedEnvVars {
			if envVar.Source == model.SourceInline {
				inlineVars[envVar.Name] = envVar.Value
			}
		}
		if inlineVars["VAR1"] != "value1" {
			t.Errorf("expected VAR1=value1, got %s", inlineVars["VAR1"])
		}
		if inlineVars["VAR2"] != "value2" {
			t.Errorf("expected VAR2=value2, got %s", inlineVars["VAR2"])
		}
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

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check that loader variable is present
		found := false
		for _, envVar := range executedEnvVars {
			if envVar.Name == "LOADER_VAR" && envVar.Value == "loader_value" && envVar.Source == model.SourceDotEnv {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected LOADER_VAR from loader to be present")
		}
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

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check that inline variable overrides loader variable
		for _, envVar := range executedEnvVars {
			if envVar.Name == "CONFLICT_VAR" {
				if envVar.Value != "inline_value" {
					t.Errorf("expected CONFLICT_VAR=inline_value, got %s", envVar.Value)
				}
				if envVar.Source != model.SourceInline {
					t.Errorf("expected source %v, got %v", model.SourceInline, envVar.Source)
				}
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

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

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
		if !foundLoader1 {
			t.Error("expected LOADER1_VAR to be present")
		}
		if !foundLoader2 {
			t.Error("expected LOADER2_VAR to be present")
		}
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

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "TEST_VAR=test_value") {
			t.Errorf("expected output to contain 'TEST_VAR=test_value', got '%s'", output)
		}
		if !strings.Contains(output, "[.env]") {
			t.Errorf("expected output to contain '[.env]', got '%s'", output)
		}
	})

	t.Run("Show environment variables with inline vars only", func(t *testing.T) {
		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		uc := usecase.NewUseCase([]loader.LoadFunc{}, executor.NewDefaultExecutor())

		// Call with inline var but no command (empty args after inline var)
		err := uc.Run(context.Background(), []string{"INLINE_VAR=inline_value"})

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "INLINE_VAR=inline_value") {
			t.Errorf("expected output to contain 'INLINE_VAR=inline_value', got '%s'", output)
		}
		if !strings.Contains(output, "[inline]") {
			t.Errorf("expected output to contain '[inline]', got '%s'", output)
		}
	})

	t.Run("Handle loader error", func(t *testing.T) {
		errorLoader := func(ctx context.Context) ([]*model.EnvVar, error) {
			return nil, os.ErrNotExist
		}

		uc := usecase.NewUseCase([]loader.LoadFunc{errorLoader}, executor.NewDefaultExecutor())

		err := uc.Run(context.Background(), []string{"echo", "test"})

		if err == nil {
			t.Error("expected error from loader")
		}
		if !strings.Contains(err.Error(), "failed to load environment variables") {
			t.Errorf("expected error message to contain 'failed to load environment variables', got %s", err.Error())
		}
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

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check inline variables
		inlineVars := make(map[string]string)
		for _, envVar := range executedEnvVars {
			if envVar.Source == model.SourceInline {
				inlineVars[envVar.Name] = envVar.Value
			}
		}

		if inlineVars["VAR_WITH_EQUALS"] != "value=with=equals" {
			t.Errorf("expected VAR_WITH_EQUALS=value=with=equals, got %s", inlineVars["VAR_WITH_EQUALS"])
		}
		if inlineVars["EMPTY_VAR"] != "" {
			t.Errorf("expected EMPTY_VAR=, got %s", inlineVars["EMPTY_VAR"])
		}
	})
}
