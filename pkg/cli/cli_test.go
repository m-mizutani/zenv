package cli_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/cli"
	"github.com/m-mizutani/zenv/v2/pkg/model"
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

	t.Run("Run with -c option", func(t *testing.T) {
		// Create temporary .yaml file
		tmpFile := gt.R1(os.CreateTemp("", "test*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `TEST_VAR:
  value: "test_value"

ANOTHER_VAR:
  value: "another_value"`

		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-c", tmpFile.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("TEST_VAR=test_value")
		gt.S(t, string(output)).Contains("ANOTHER_VAR=another_value")
		gt.S(t, string(output)).Contains("[.yaml]")
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

	t.Run("Run with both -e and -c options", func(t *testing.T) {
		// Create .env file
		envFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(envFile.Name())

		envContent := `ENV_VAR=env_value`
		gt.R1(envFile.WriteString(envContent)).NoError(t)
		envFile.Close()

		// Create .yaml file
		yamlFile := gt.R1(os.CreateTemp("", "test*.yaml")).NoError(t)
		defer os.Remove(yamlFile.Name())

		yamlContent := `YAML_VAR:
  value: "yaml_value"`
		gt.R1(yamlFile.WriteString(yamlContent)).NoError(t)
		yamlFile.Close()

		// Capture stdout
		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-e", envFile.Name(), "-c", yamlFile.Name()}
		err := cli.Run(context.Background(), args)

		// Restore stdout and read captured output
		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("ENV_VAR=env_value")
		gt.S(t, string(output)).Contains("YAML_VAR=yaml_value")
		gt.S(t, string(output)).Contains("[.env]")
		gt.S(t, string(output)).Contains("[.yaml]")
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

	t.Run("Run with -c option (.hcl file)", func(t *testing.T) {
		tmpFile := gt.R1(os.CreateTemp("", "test*.hcl")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `TEST_VAR = "test_value"

ANOTHER_VAR {
  value = "another_value"
}
`
		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-c", tmpFile.Name()}
		err := cli.Run(context.Background(), args)

		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("TEST_VAR=test_value")
		gt.S(t, string(output)).Contains("ANOTHER_VAR=another_value")
	})

	t.Run("Default path: .env.hcl takes precedence over .env.yaml", func(t *testing.T) {
		tmpDir := t.TempDir()

		hclContent := `FROM_HCL = "hcl_wins"
`
		gt.NoError(t, os.WriteFile(tmpDir+"/.env.hcl", []byte(hclContent), 0600))

		yamlContent := `FROM_YAML:
  value: "yaml_value"
FROM_HCL:
  value: "yaml_loses"
`
		gt.NoError(t, os.WriteFile(tmpDir+"/.env.yaml", []byte(yamlContent), 0600))

		oldWd := gt.R1(os.Getwd()).NoError(t)
		gt.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(oldWd) }()

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv"}
		err := cli.Run(context.Background(), args)

		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		// HCL value wins
		gt.S(t, string(output)).Contains("FROM_HCL=hcl_wins")
		// YAML is fully ignored
		gt.S(t, string(output)).NotContains("FROM_YAML=yaml_value")
		gt.S(t, string(output)).NotContains("hcl_loses")
	})

	t.Run("Explicit -e with missing file errors out", func(t *testing.T) {
		args := []string{"zenv", "-e", "/definitely/not/here.env"}
		err := cli.Run(context.Background(), args)

		gt.Error(t, err)
		var cfgErr *model.ConfigFileError
		gt.True(t, errors.As(err, &cfgErr))
		gt.Equal(t, cfgErr.Format, model.FormatDotEnv)
		gt.Equal(t, cfgErr.Reason, model.ReasonNotFound)
	})

	t.Run("Explicit -c with missing YAML file errors out", func(t *testing.T) {
		args := []string{"zenv", "-c", "/definitely/not/here.yaml"}
		err := cli.Run(context.Background(), args)

		gt.Error(t, err)
		var cfgErr *model.ConfigFileError
		gt.True(t, errors.As(err, &cfgErr))
		gt.Equal(t, cfgErr.Format, model.FormatYAML)
		gt.Equal(t, cfgErr.Reason, model.ReasonNotFound)
	})

	t.Run("Explicit -c with missing HCL file errors out", func(t *testing.T) {
		args := []string{"zenv", "-c", "/definitely/not/here.hcl"}
		err := cli.Run(context.Background(), args)

		gt.Error(t, err)
		var cfgErr *model.ConfigFileError
		gt.True(t, errors.As(err, &cfgErr))
		gt.Equal(t, cfgErr.Format, model.FormatHCL)
		gt.Equal(t, cfgErr.Reason, model.ReasonNotFound)
	})

	t.Run("Invalid log level value errors out", func(t *testing.T) {
		args := []string{"zenv", "-l", "noisy"}
		err := cli.Run(context.Background(), args)

		gt.Error(t, err)
		var lvlErr *model.InvalidLogLevelError
		gt.True(t, errors.As(err, &lvlErr))
		gt.Equal(t, lvlErr.Value, "noisy")
	})

	t.Run("Unknown profile errors out", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "config.yaml")
		content := `API_URL:
  value: "https://api.example.com"
  profile:
    dev:
      value: "http://localhost"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(content), 0600))

		args := []string{"zenv", "-c", yamlPath, "-p", "prod"}
		err := cli.Run(context.Background(), args)

		gt.Error(t, err)
		var pe *model.ProfileNotFoundError
		gt.True(t, errors.As(err, &pe))
		gt.Equal(t, pe.Profile, "prod")
		gt.Equal(t, len(pe.Available), 1)
		gt.Equal(t, pe.Available[0], "dev")
	})

	t.Run("Known profile succeeds preflight", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "config.yaml")
		content := `API_URL:
  value: "default"
  profile:
    dev:
      value: "dev-url"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(content), 0600))

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv", "-c", yamlPath, "-p", "dev"}
		err := cli.Run(context.Background(), args)

		w.Close()
		os.Stdout = oldStdout
		output := gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
		gt.S(t, string(output)).Contains("API_URL=dev-url")
	})

	t.Run("Default .env probe still tolerates absence", func(t *testing.T) {
		// In a temp directory with no .env, .env.yaml, .env.hcl, running zenv
		// with no flags must NOT fail just because nothing was discovered.
		tmpDir := t.TempDir()
		oldWd := gt.R1(os.Getwd()).NoError(t)
		gt.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(oldWd) }()

		r, w, _ := os.Pipe()
		oldStdout := os.Stdout
		os.Stdout = w

		args := []string{"zenv"}
		err := cli.Run(context.Background(), args)

		w.Close()
		os.Stdout = oldStdout
		gt.R1(io.ReadAll(r)).NoError(t)

		gt.NoError(t, err)
	})
}

func TestIsTruthyEnv(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"no":    false,
		"off":   false,
		"1":     true,
		"true":  true,
		"TRUE":  true,
		"True":  true,
		"yes":   true,
		"on":    true,
		" 1 ":   true,
		"foo":   false,
	}
	for input, want := range cases {
		got := cli.IsTruthyEnvForTest(input)
		gt.Equal(t, got, want)
	}
}

func TestParseLogLevel(t *testing.T) {
	t.Run("known values map to slog levels", func(t *testing.T) {
		cases := map[string]slog.Level{
			"debug":   slog.LevelDebug,
			"DEBUG":   slog.LevelDebug,
			"info":    slog.LevelInfo,
			"warn":    slog.LevelWarn,
			"warning": slog.LevelWarn,
			"error":   slog.LevelError,
		}
		for in, want := range cases {
			got, err := cli.ParseLogLevel(in)
			gt.NoError(t, err)
			gt.Equal(t, got, want)
		}
	})

	t.Run("unknown value returns InvalidLogLevelError", func(t *testing.T) {
		_, err := cli.ParseLogLevel("verbose")
		gt.Error(t, err)
		var lvlErr *model.InvalidLogLevelError
		gt.True(t, errors.As(err, &lvlErr))
		gt.Equal(t, lvlErr.Value, "verbose")
	})

	t.Run("empty string is rejected", func(t *testing.T) {
		_, err := cli.ParseLogLevel("")
		gt.Error(t, err)
		var lvlErr *model.InvalidLogLevelError
		gt.True(t, errors.As(err, &lvlErr))
	})
}
