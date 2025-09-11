package loader_test

import (
	"context"
	"strings"
	"testing"

	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestTOMLLoader(t *testing.T) {

	t.Run("Load valid TOML file with static values", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/valid.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if len(envVars) != 3 {
			t.Errorf("expected 3 env vars, got %d", len(envVars))
		}

		// Verify values
		expected := map[string]string{
			"DATABASE_URL":     "postgresql://localhost/testdb",
			"API_KEY":          "secret123",
			"MULTILINE_CONFIG": "line1\nline2\nline3",
		}

		for _, envVar := range envVars {
			if envVar.Source != model.SourceTOML {
				t.Errorf("expected source %v, got %v", model.SourceTOML, envVar.Source)
			}
			expectedValue, exists := expected[envVar.Name]
			if !exists {
				t.Errorf("unexpected key: %s", envVar.Name)
			}
			if envVar.Value != expectedValue {
				t.Errorf("expected value %s, got %s", expectedValue, envVar.Value)
			}
		}
	})

	t.Run("Load TOML file with file reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/with_file.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if len(envVars) != 1 {
			t.Errorf("expected 1 env var, got %d", len(envVars))
		}
		if envVars[0].Name != "CONFIG_DATA" {
			t.Errorf("expected name CONFIG_DATA, got %s", envVars[0].Name)
		}
		if envVars[0].Value != "config file content" {
			t.Errorf("expected value 'config file content', got %s", envVars[0].Value)
		}
		if envVars[0].Source != model.SourceTOML {
			t.Errorf("expected source %v, got %v", model.SourceTOML, envVars[0].Source)
		}
	})

	t.Run("Load TOML file with command execution", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/with_command.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if len(envVars) != 1 {
			t.Errorf("expected 1 env var, got %d", len(envVars))
		}
		if envVars[0].Name != "HOSTNAME" {
			t.Errorf("expected name HOSTNAME, got %s", envVars[0].Name)
		}
		if envVars[0].Value != "test-host" {
			t.Errorf("expected value 'test-host', got %s", envVars[0].Value)
		}
		if envVars[0].Source != model.SourceTOML {
			t.Errorf("expected source %v, got %v", model.SourceTOML, envVars[0].Source)
		}
	})

	t.Run("Handle non-existent file", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/non_existent.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if envVars != nil {
			t.Errorf("expected nil envVars, got %v", envVars)
		}
	})

	t.Run("Handle invalid TOML syntax", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/invalid_syntax.toml")
		_, err := loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for invalid TOML syntax")
		}
		if !strings.Contains(err.Error(), "failed to parse TOML file") {
			t.Errorf("expected error to contain 'failed to parse TOML file', got %s", err.Error())
		}
	})

	t.Run("Handle validation errors", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/validation_error.toml")
		_, err := loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for validation")
		}
		if !strings.Contains(err.Error(), "multiple value types specified") {
			t.Errorf("expected error to contain 'multiple value types specified', got %s", err.Error())
		}
	})

	t.Run("Handle missing file reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/missing_file.toml")
		_, err := loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for missing file")
		}
		if !strings.Contains(err.Error(), "failed to read file") {
			t.Errorf("expected error to contain 'failed to read file', got %s", err.Error())
		}
	})

	t.Run("Handle command execution failure", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/command_failure.toml")
		_, err := loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for command execution failure")
		}
		// The 'false' command exits with non-zero status but doesn't produce an error
		// So we check for the empty output case
	})

	t.Run("Load TOML file with alias", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/alias.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}

		// Check that alias resolves correctly
		aliasVars := make(map[string]string)
		for _, envVar := range envVars {
			aliasVars[envVar.Name] = envVar.Value
		}

		// DATABASE_URL should resolve to the value of PRIMARY_DB
		if aliasVars["DATABASE_URL"] != aliasVars["PRIMARY_DB"] {
			t.Errorf("DATABASE_URL should resolve to PRIMARY_DB value, got %s", aliasVars["DATABASE_URL"])
		}

		// BACKUP_DATABASE should resolve to the value of SECONDARY_DB
		if aliasVars["BACKUP_DATABASE"] != aliasVars["SECONDARY_DB"] {
			t.Errorf("BACKUP_DATABASE should resolve to SECONDARY_DB value, got %s", aliasVars["BACKUP_DATABASE"])
		}
	})

	t.Run("Alias resolves system environment variable", func(t *testing.T) {
		// Set a system environment variable for testing
		t.Setenv("TEST_SYSTEM_VAR", "system_value")

		loadFunc := loader.NewTOMLLoader("testdata/alias_system.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}

		// Find the MY_VAR environment variable
		var myVarValue string
		for _, envVar := range envVars {
			if envVar.Name == "MY_VAR" {
				myVarValue = envVar.Value
				break
			}
		}

		if myVarValue != "system_value" {
			t.Errorf("MY_VAR should resolve to system environment variable value 'system_value', got %s", myVarValue)
		}
	})

	t.Run("Alias with non-existent target returns empty string", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/alias_missing.toml")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}

		// Find the MISSING_ALIAS environment variable
		var missingValue string
		for _, envVar := range envVars {
			if envVar.Name == "MISSING_ALIAS" {
				missingValue = envVar.Value
				break
			}
		}

		if missingValue != "" {
			t.Errorf("MISSING_ALIAS should resolve to empty string when target doesn't exist, got %s", missingValue)
		}
	})

	t.Run("Handle circular alias reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/circular_alias.toml")
		_, err := loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for circular alias reference")
		}
		if !strings.Contains(err.Error(), "circular alias reference") {
			t.Errorf("expected error to contain 'circular alias reference', got %s", err.Error())
		}
	})

	t.Run("Validation error when multiple types including alias", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/alias_multiple_types.toml")
		_, err := loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for multiple value types")
		}
		if !strings.Contains(err.Error(), "multiple value types specified") {
			t.Errorf("expected error to contain 'multiple value types specified', got %s", err.Error())
		}
	})
}
