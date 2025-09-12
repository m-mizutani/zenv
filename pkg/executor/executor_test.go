package executor_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestDefaultExecutor(t *testing.T) {

	t.Run("Execute simple command successfully", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
		}

		err := execFunc(context.Background(), "echo", []string{"hello"}, envVars)
		gt.NoError(t, err)
	})

	t.Run("Execute command with environment variables", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
			{Name: "ANOTHER_VAR", Value: "another_value", Source: model.SourceDotEnv},
		}

		// Test with a command that uses environment variables
		err := execFunc(context.Background(), "sh", []string{"-c", "test -n \"$TEST_VAR\""}, envVars)
		gt.NoError(t, err)
	})

	t.Run("Handle command that returns non-zero exit code", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with a command that exits with code 1
		err := execFunc(context.Background(), "sh", []string{"-c", "exit 1"}, envVars)
		gt.Error(t, err)
		exitCode := model.GetExitCode(err)
		gt.Equal(t, exitCode, 1)
	})

	t.Run("Handle command that returns different exit code", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with a command that exits with code 42
		err := execFunc(context.Background(), "sh", []string{"-c", "exit 42"}, envVars)
		gt.Error(t, err)
		exitCode := model.GetExitCode(err)
		gt.Equal(t, exitCode, 42)
	})

	t.Run("Handle non-existent command", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		err := execFunc(context.Background(), "nonexistentcommand123", []string{}, envVars)
		gt.Error(t, err)
		exitCode := model.GetExitCode(err)
		gt.Equal(t, exitCode, 1) // Should return default exit code 1 for command not found
	})

	t.Run("Execute command with empty arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
		}

		err := execFunc(context.Background(), "true", []string{}, envVars)
		gt.NoError(t, err)
	})

	t.Run("Pass through stdout and stderr", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Command that produces output on stdout
		err := execFunc(context.Background(), "echo", []string{"test output"}, envVars)
		gt.NoError(t, err)

		// Command that produces output on stderr
		err = execFunc(context.Background(), "sh", []string{"-c", ">&2 echo error output; exit 0"}, envVars)
		gt.NoError(t, err)
	})

	t.Run("Execute command with multiple arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		err := execFunc(context.Background(), "echo", []string{"arg1", "arg2", "arg3"}, envVars)
		gt.NoError(t, err)
	})

	t.Run("Execute command with special characters in arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with various special characters
		args := []string{"$VAR", "\"quoted\"", "'single'", "space test", "new\nline"}
		err := execFunc(context.Background(), "echo", args, envVars)
		gt.NoError(t, err)
	})

	t.Run("Environment variables are properly set", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "CUSTOM_VAR", Value: "custom_value", Source: model.SourceInline},
		}

		// Check if the environment variable is accessible
		err := execFunc(context.Background(), "sh", []string{"-c", "[ \"$CUSTOM_VAR\" = \"custom_value\" ]"}, envVars)
		gt.NoError(t, err)
	})
}
