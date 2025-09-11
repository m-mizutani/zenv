package cli_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/zenv/pkg/cli"
)

func TestCLI(t *testing.T) {

	t.Run("Run with -e option", func(t *testing.T) {
		// Create temporary .env file
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `TEST_VAR=test_value
ANOTHER_VAR=another_value`

		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", tmpFile.Name()}
		err = cli.Run(context.Background(), args)

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
		if !strings.Contains(output, "ANOTHER_VAR=another_value") {
			t.Errorf("expected output to contain 'ANOTHER_VAR=another_value', got '%s'", output)
		}
		if !strings.Contains(output, "[.env]") {
			t.Errorf("expected output to contain '[.env]', got '%s'", output)
		}
	})

	t.Run("Run with -t option", func(t *testing.T) {
		// Create temporary .toml file
		tmpFile, err := os.CreateTemp("", "test*.toml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `[TEST_VAR]
value = "test_value"

[ANOTHER_VAR]
value = "another_value"`

		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-t", tmpFile.Name()}
		err = cli.Run(context.Background(), args)

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
		if !strings.Contains(output, "ANOTHER_VAR=another_value") {
			t.Errorf("expected output to contain 'ANOTHER_VAR=another_value', got '%s'", output)
		}
		if !strings.Contains(output, "[.toml]") {
			t.Errorf("expected output to contain '[.toml]', got '%s'", output)
		}
	})

	t.Run("Run with multiple -e options", func(t *testing.T) {
		// Create first .env file
		tmpFile1, err := os.CreateTemp("", "test1*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile1.Name())

		content1 := `VAR1=value1`
		_, err = tmpFile1.WriteString(content1)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile1.Close()

		// Create second .env file
		tmpFile2, err := os.CreateTemp("", "test2*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile2.Name())

		content2 := `VAR2=value2`
		_, err = tmpFile2.WriteString(content2)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile2.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", tmpFile1.Name(), "-e", tmpFile2.Name()}
		err = cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "VAR1=value1") {
			t.Errorf("expected output to contain 'VAR1=value1', got '%s'", output)
		}
		if !strings.Contains(output, "VAR2=value2") {
			t.Errorf("expected output to contain 'VAR2=value2', got '%s'", output)
		}
	})

	t.Run("Run with both -e and -t options", func(t *testing.T) {
		// Create .env file
		envFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(envFile.Name())

		envContent := `ENV_VAR=env_value`
		_, err = envFile.WriteString(envContent)
		if err != nil {
			t.Fatal(err)
		}
		envFile.Close()

		// Create .toml file
		tomlFile, err := os.CreateTemp("", "test*.toml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tomlFile.Name())

		tomlContent := `[TOML_VAR]
value = "toml_value"`
		_, err = tomlFile.WriteString(tomlContent)
		if err != nil {
			t.Fatal(err)
		}
		tomlFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", envFile.Name(), "-t", tomlFile.Name()}
		err = cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "ENV_VAR=env_value") {
			t.Errorf("expected output to contain 'ENV_VAR=env_value', got '%s'", output)
		}
		if !strings.Contains(output, "TOML_VAR=toml_value") {
			t.Errorf("expected output to contain 'TOML_VAR=toml_value', got '%s'", output)
		}
		if !strings.Contains(output, "[.env]") {
			t.Errorf("expected output to contain '[.env]', got '%s'", output)
		}
		if !strings.Contains(output, "[.toml]") {
			t.Errorf("expected output to contain '[.toml]', got '%s'", output)
		}
	})

	t.Run("Run command execution", func(t *testing.T) {
		// Create temporary .env file
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `TEST_VAR=hello_world`
		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Test with a simple command
		args := []string{"zenv", "-e", tmpFile.Name(), "echo", "test"}
		err = cli.Run(context.Background(), args)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("List mode with no command", func(t *testing.T) {
		// Create temporary .env file
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `LIST_VAR=list_value`
		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		// Run with no command (should trigger list mode)
		args := []string{"zenv", "-e", tmpFile.Name()}
		err = cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.Contains(output, "LIST_VAR=list_value") {
			t.Errorf("expected output to contain 'LIST_VAR=list_value', got '%s'", output)
		}
	})

	t.Run("Handle non-existent file gracefully", func(t *testing.T) {
		// This should not error as non-existent files return nil, nil
		args := []string{"zenv", "-e", "non_existent.env"}
		err := cli.Run(context.Background(), args)

		if err != nil {
			t.Fatalf("expected no error for non-existent file, got %v", err)
		}
	})
}
