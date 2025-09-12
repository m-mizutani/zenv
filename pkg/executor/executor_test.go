package executor_test

import (
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

		exitCode := gt.R1(execFunc("echo", []string{"hello"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})

	t.Run("Execute command with environment variables", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
			{Name: "ANOTHER_VAR", Value: "another_value", Source: model.SourceDotEnv},
		}

		// Test with a command that uses environment variables
		exitCode := gt.R1(execFunc("sh", []string{"-c", "test -n \"$TEST_VAR\""}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})

	t.Run("Handle command that returns non-zero exit code", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with a command that exits with code 1
		exitCode := gt.R1(execFunc("sh", []string{"-c", "exit 1"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 1)
	})

	t.Run("Handle command that returns different exit code", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with a command that exits with code 42
		exitCode := gt.R1(execFunc("sh", []string{"-c", "exit 42"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 42)
	})

	t.Run("Handle non-existent command", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		_, err := execFunc("nonexistentcommand123", []string{}, envVars)
		gt.Error(t, err)
	})

	t.Run("Execute command with empty arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "TEST_VAR", Value: "test_value", Source: model.SourceSystem},
		}

		exitCode := gt.R1(execFunc("true", []string{}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})

	t.Run("Pass through stdout and stderr", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Command that produces output on stdout
		exitCode := gt.R1(execFunc("echo", []string{"test output"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)

		// Command that produces output on stderr
		exitCode = gt.R1(execFunc("sh", []string{"-c", ">&2 echo error output; exit 0"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})

	t.Run("Execute command with multiple arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		exitCode := gt.R1(execFunc("echo", []string{"arg1", "arg2", "arg3"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})

	t.Run("Execute command with special characters in arguments", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{}

		// Test with various special characters
		args := []string{"$VAR", "\"quoted\"", "'single'", "space test", "new\nline"}
		exitCode := gt.R1(execFunc("echo", args, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})

	t.Run("Environment variables are properly set", func(t *testing.T) {
		execFunc := executor.NewDefaultExecutor()
		envVars := []*model.EnvVar{
			{Name: "CUSTOM_VAR", Value: "custom_value", Source: model.SourceInline},
		}

		// Check if the environment variable is accessible
		exitCode := gt.R1(execFunc("sh", []string{"-c", "[ \"$CUSTOM_VAR\" = \"custom_value\" ]"}, envVars)).NoError(t)
		gt.Equal(t, exitCode, 0)
	})
}