package cli_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/cli"
)

func TestCLI(t *testing.T) {

	t.Run("Run with -e option", func(t *testing.T) {
		// Create temporary .env file
		tmpFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `TEST_VAR=test_value
ANOTHER_VAR=another_value`

		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", tmpFile.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("TEST_VAR=test_value")
		gt.S(t, string(output)).Contains("ANOTHER_VAR=another_value")
		gt.S(t, string(output)).Contains("[.env]")
	})

	t.Run("Run with -t option", func(t *testing.T) {
		// Create temporary .toml file
		tmpFile := gt.R1(os.CreateTemp("", "test*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `[TEST_VAR]
value = "test_value"

[ANOTHER_VAR]
value = "another_value"`

		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-t", tmpFile.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("TEST_VAR=test_value")
		gt.S(t, string(output)).Contains("ANOTHER_VAR=another_value")
		gt.S(t, string(output)).Contains("[.toml]")
	})

	t.Run("Run with multiple -e options", func(t *testing.T) {
		// Create first .env file
		tmpFile1 := gt.R1(os.CreateTemp("", "test1*.env")).NoError(t)
		defer os.Remove(tmpFile1.Name())

		content1 := `VAR1=value1`
		gt.R1(tmpFile1.WriteString(content1)).NoError(t)
		tmpFile1.Close()

		// Create second .env file
		tmpFile2 := gt.R1(os.CreateTemp("", "test2*.env")).NoError(t)
		defer os.Remove(tmpFile2.Name())

		content2 := `VAR2=value2`
		gt.R1(tmpFile2.WriteString(content2)).NoError(t)
		tmpFile2.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", tmpFile1.Name(), "-e", tmpFile2.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("VAR1=value1")
		gt.S(t, string(output)).Contains("VAR2=value2")
	})

	t.Run("Run with both -e and -t options", func(t *testing.T) {
		// Create .env file
		envFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(envFile.Name())

		envContent := `ENV_VAR=env_value`
		gt.R1(envFile.WriteString(envContent)).NoError(t)
		envFile.Close()

		// Create .toml file
		tomlFile := gt.R1(os.CreateTemp("", "test*.toml")).NoError(t)
		defer os.Remove(tomlFile.Name())

		tomlContent := `[TOML_VAR]
value = "toml_value"`
		gt.R1(tomlFile.WriteString(tomlContent)).NoError(t)
		tomlFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", envFile.Name(), "-t", tomlFile.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("ENV_VAR=env_value")
		gt.S(t, string(output)).Contains("TOML_VAR=toml_value")
		gt.S(t, string(output)).Contains("[.env]")
		gt.S(t, string(output)).Contains("[.toml]")
	})

	t.Run("Run command execution", func(t *testing.T) {
		// Create temporary .env file
		tmpFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `TEST_VAR=hello_world`
		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Test with a simple command
		args := []string{"zenv", "-e", tmpFile.Name(), "echo", "test"}
		err := cli.Run(context.Background(), args)

		gt.NoError(t, err)
	})

	t.Run("List mode with no command", func(t *testing.T) {
		// Create temporary .env file
		tmpFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `LIST_VAR=list_value`
		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		// Run with no command (should trigger list mode)
		args := []string{"zenv", "-e", tmpFile.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("LIST_VAR=list_value")
	})

	t.Run("Handle non-existent file gracefully", func(t *testing.T) {
		// Capture stdout to prevent flooding test output
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		// This should not error as non-existent files return nil, nil
		args := []string{"zenv", "-e", "non_existent.env"}
		err := cli.Run(context.Background(), args)

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout
		// Read and discard output
		gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
	})
}
