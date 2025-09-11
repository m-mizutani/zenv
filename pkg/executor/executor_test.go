package executor_test

import (
	"testing"

	"github.com/m-mizutani/zenv/pkg/executor"
	"github.com/m-mizutani/zenv/pkg/model"
)

func TestDefaultExecutor(t *testing.T) {

	t.Run("Execute simple command successfully", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
		}

		exitCode, err := execFunc("echo", []string{"hello"}, envVars)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("Execute command with environment variables", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
			{Name: "ANOTHER_VAR", Value: "another_value", Source: model.SourceDotEnv},
		}

		// Test with a command that uses environment variables
		exitCode, err := execFunc("sh", []string{"-c", "test -n \"$TEST_VAR\""}, envVars)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("Handle command that returns non-zero exit code", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with a command that exits with code 1
		exitCode, err := execFunc("sh", []string{"-c", "exit 1"}, envVars)

		if err != nil {
			t.Fatalf("expected no error for non-zero exit, got %v", err)
		}
		if exitCode != 1 {
			t.Errorf("expected exit code 1, got %d", exitCode)
		}
	})

	t.Run("Handle command that returns different exit code", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with a command that exits with code 42
		exitCode, err := execFunc("sh", []string{"-c", "exit 42"}, envVars)

		if err != nil {
			t.Fatalf("expected no error for non-zero exit, got %v", err)
		}
		if exitCode != 42 {
			t.Errorf("expected exit code 42, got %d", exitCode)
		}
	})

	t.Run("Handle non-existent command", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		_, err := execFunc("nonexistentcommand123", []string{}, envVars)

		if err == nil {
			t.Error("expected error for non-existent command")
		}
	})

	t.Run("Execute command with empty arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
		}

		exitCode, err := execFunc("true", []string{}, envVars)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("Execute command with multiple arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		exitCode, err := execFunc("test", []string{"-n", "hello"}, envVars)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})

	t.Run("Execute command with no environment variables", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		exitCode, err := execFunc("echo", []string{"hello"}, envVars)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
	})
}
