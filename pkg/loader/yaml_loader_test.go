package loader_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestYAMLLoader(t *testing.T) {

	t.Run("Load valid YAML file with static values", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/valid.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 3)

		// Verify values
		expected := map[string]string{
			"DATABASE_URL":     "postgresql://localhost/testdb",
			"API_KEY":          "secret123",
			"MULTILINE_CONFIG": "line1\nline2\nline3",
		}

		for _, envVar := range envVars {
			gt.Equal(t, envVar.Source, model.SourceYAML)
			expectedValue, exists := expected[envVar.Name]
			gt.True(t, exists)
			gt.Equal(t, envVar.Value, expectedValue)
		}
	})

	t.Run("Load YAML file with file reference", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/with_file.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "CONFIG_DATA")
		gt.Equal(t, envVars[0].Value, "config file content")
		gt.Equal(t, envVars[0].Source, model.SourceYAML)
	})

	t.Run("Load YAML file with command execution", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/with_command.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "HOSTNAME")
		gt.Equal(t, envVars[0].Value, "test-host")
		gt.Equal(t, envVars[0].Source, model.SourceYAML)
	})

	t.Run("Handle non-existent file", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/non_existent.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		gt.Nil(t, envVars)
	})

	t.Run("Handle invalid YAML syntax", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/invalid_syntax.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to parse YAML file")
	})

	t.Run("Handle validation errors", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/validation_error.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	t.Run("Handle missing file reference", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/missing_file.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to read file")
	})

	t.Run("Handle command execution failure", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/command_failure.yaml")
		_, err := loadFunc(context.Background())
		// The 'false' command exits with non-zero status
		gt.Error(t, err)
	})

	t.Run("Load YAML file with alias", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/alias.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Check that alias resolves correctly
		aliasVars := make(map[string]string)
		for _, envVar := range envVars {
			aliasVars[envVar.Name] = envVar.Value
		}

		// DATABASE_URL should resolve to the value of PRIMARY_DB
		gt.Equal(t, aliasVars["DATABASE_URL"], aliasVars["PRIMARY_DB"])
		// BACKUP_DATABASE should resolve to the value of SECONDARY_DB
		gt.Equal(t, aliasVars["BACKUP_DATABASE"], aliasVars["SECONDARY_DB"])
	})

	t.Run("Alias resolves system environment variable", func(t *testing.T) {
		// Set a system environment variable for testing
		t.Setenv("TEST_SYSTEM_VAR", "system_value")

		loadFunc := loader.NewYAMLLoader("testdata/alias_system.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Find the MY_VAR environment variable
		var myVarValue string
		for _, envVar := range envVars {
			if envVar.Name == "MY_VAR" {
				myVarValue = envVar.Value
				break
			}
		}

		gt.Equal(t, myVarValue, "system_value")
	})

	t.Run("Alias precedence: YAML overrides system environment", func(t *testing.T) {
		// Set a system environment variable
		t.Setenv("SHARED_VAR", "system_value")

		// Create a YAML file that defines the same variable and an alias to it
		yamlContent := `
SHARED_VAR:
  value: "yaml_value"

ALIAS_TO_SHARED:
  alias: "SHARED_VAR"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_precedence*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Find the ALIAS_TO_SHARED variable
		var aliasValue string
		for _, envVar := range envVars {
			if envVar.Name == "ALIAS_TO_SHARED" {
				aliasValue = envVar.Value
				break
			}
		}

		// The alias should resolve to the YAML value, not the system value
		gt.Equal(t, aliasValue, "yaml_value")
	})

	t.Run("Alias resolves empty system environment variable", func(t *testing.T) {
		// Set an empty system environment variable
		t.Setenv("EMPTY_SYSTEM_VAR", "")

		yamlContent := `
ALIAS_TO_EMPTY:
  alias: "EMPTY_SYSTEM_VAR"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_empty*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Find the ALIAS_TO_EMPTY variable
		found := false
		var aliasValue string
		for _, envVar := range envVars {
			if envVar.Name == "ALIAS_TO_EMPTY" {
				found = true
				aliasValue = envVar.Value
				break
			}
		}

		gt.True(t, found)
		gt.Equal(t, aliasValue, "")
	})

	t.Run("Alias with non-existent target returns error", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/alias_missing.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Handle circular alias reference", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/circular_alias.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Validation error when multiple types including alias", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/alias_multiple_types.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	// Template tests
	t.Run("Load YAML file with basic template", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/template.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Check that template resolves correctly
		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// AUTH_HEADER should be "Bearer secret-token-123"
		gt.Equal(t, vars["AUTH_HEADER"], "Bearer secret-token-123")

		// DATABASE_URL should be properly formatted
		expectedDBURL := "postgresql://admin:secret@localhost:5432/myapp"
		gt.Equal(t, vars["DATABASE_URL"], expectedDBURL)

		// CONNECTION_STRING should resolve alias through template
		gt.Equal(t, vars["CONNECTION_STRING"], "Connection: postgres://primary.db.example.com/app")
	})

	t.Run("Template with conditional logic", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/template.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// API_ENDPOINT should use staging URL when USE_STAGING is true
		gt.Equal(t, vars["API_ENDPOINT"], "https://staging.api.example.com")
	})

	t.Run("Template with system environment variable", func(t *testing.T) {
		// Set a system environment variable for testing
		t.Setenv("TEST_SYS_VAR", "system_value")

		// Create a YAML file that uses system var in template
		yamlContent := `
FROM_SYSTEM:
  value: "System: {{ .TEST_SYS_VAR }}"
  refs: ["TEST_SYS_VAR"]
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_sys*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Find the FROM_SYSTEM variable
		var fromSystemValue string
		for _, envVar := range envVars {
			if envVar.Name == "FROM_SYSTEM" {
				fromSystemValue = envVar.Value
				break
			}
		}

		gt.Equal(t, fromSystemValue, "System: system_value")
	})

	t.Run("Template with empty reference", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/template_complex.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// WITH_EMPTY should have empty brackets
		gt.Equal(t, vars["WITH_EMPTY"], "Value: []")

		// MISSING_REF_TEMPLATE should not exist anymore (removed from template_complex.yaml)
		_, exists := vars["MISSING_REF_TEMPLATE"]
		gt.False(t, exists)
	})

	t.Run("Template with nested references", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/template_complex.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// API_PATH should combine BASE_URL and API_VERSION
		gt.Equal(t, vars["API_PATH"], "https://api.example.com/v2")

		// FULL_ENDPOINT should reference API_PATH
		gt.Equal(t, vars["FULL_ENDPOINT"], "https://api.example.com/v2/users")

		// EMAIL should combine USERNAME and DOMAIN
		gt.Equal(t, vars["EMAIL"], "john_doe@example.com")
	})

	t.Run("Template with complex logic", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/template_complex.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// LOG_CONFIG should be "debug" when DEBUG_MODE is true
		gt.Equal(t, vars["LOG_CONFIG"], "debug")
	})

	t.Run("Handle circular template reference", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/template_circular.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Template syntax error", func(t *testing.T) {
		// Create a YAML file with template syntax error
		yamlContent := `
SYNTAX_ERROR:
  value: "{{ .VAR_NAME"
  refs: ["VAR_NAME"]

VAR_NAME:
  value: "test"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_syntax*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to parse value template")
	})

	t.Run("Template without refs field", func(t *testing.T) {
		// Create a YAML file with value but no refs (should work as static value)
		yamlContent := `
NO_REFS:
  value: "Static template text"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_norefs*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "NO_REFS")
		gt.Equal(t, envVars[0].Value, "Static template text")
	})

	t.Run("Refs without template field", func(t *testing.T) {
		// Create a YAML file with refs but no template
		yamlContent := `
REFS_ONLY:
  refs: ["SOME_VAR"]
`
		tmpFile := gt.R1(os.CreateTemp("", "test_refs_only*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("refs can only be used with value or command")
	})

	t.Run("Template with multiple value types", func(t *testing.T) {
		// Create a YAML file with both value and file (invalid)
		yamlContent := `
MULTIPLE_TYPES:
  value: "some value"
  file: "/path/to/file"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_multiple*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	t.Run("Template referenced by an alias", func(t *testing.T) {
		yamlContent := `
TEMPLATE_VAR:
  value: "hello world"
  refs: []

ALIAS_VAR:
  alias: "TEMPLATE_VAR"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_alias*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// Both TEMPLATE_VAR and ALIAS_VAR should resolve to "hello world"
		gt.Equal(t, vars["TEMPLATE_VAR"], "hello world")
		gt.Equal(t, vars["ALIAS_VAR"], "hello world")
	})

	t.Run("Alias pointing to template variable with refs", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/alias_to_template.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// TEMPLATE_VAR should resolve to "hello world"
		gt.Equal(t, vars["TEMPLATE_VAR"], "hello world")

		// ALIAS_TO_TEMPLATE should also resolve to "hello world"
		gt.Equal(t, vars["ALIAS_TO_TEMPLATE"], "hello world")

		// ALIAS_TO_ALIAS should also resolve to "hello world"
		gt.Equal(t, vars["ALIAS_TO_ALIAS"], "hello world")
	})

	t.Run("Handle complex circular references (alias->template->alias)", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/complex_circular.yaml")
		_, err := loadFunc(context.Background())

		// Should detect circular reference in any of the complex cases
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Circular reference via alias and template mix", func(t *testing.T) {
		// Create a specific test case: A->B(template)->C->A
		yamlContent := `
VAR_A:
  alias: "VAR_B"

VAR_B:
  value: "B uses {{ .VAR_C }}"
  refs: ["VAR_C"]

VAR_C:
  alias: "VAR_A"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_circular_mix*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Template self-reference through alias", func(t *testing.T) {
		// Create a test where template references itself through an alias
		yamlContent := `
SELF_TEMPLATE:
  value: "Self: {{ .SELF_ALIAS }}"
  refs: ["SELF_ALIAS"]

SELF_ALIAS:
  alias: "SELF_TEMPLATE"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_self_ref_mix*.yaml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(yamlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewYAMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	// Cross-reference tests: YAML can reference .env and system variables
	t.Run("Template can reference .env variables", func(t *testing.T) {
		// Create temporary YAML file with template
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
DB_URL:
  value: "postgres://{{ .DB_USER }}:{{ .DB_PASS }}@{{ .DB_HOST }}:5432/mydb"
  refs: ["DB_USER", "DB_PASS", "DB_HOST"]
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create existing variables (as if from .env file)
		existingVars := []*model.EnvVar{
			{Name: "DB_USER", Value: "admin", Source: model.SourceDotEnv},
			{Name: "DB_PASS", Value: "secret123", Source: model.SourceDotEnv},
			{Name: "DB_HOST", Value: "localhost", Source: model.SourceDotEnv},
		}

		// Create YAML loader with existing variables
		loader := loader.NewYAMLLoader(yamlPath, existingVars)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "DB_URL")
		gt.Equal(t, vars[0].Value, "postgres://admin:secret123@localhost:5432/mydb")
	})

	t.Run("Alias can reference .env variables", func(t *testing.T) {
		// Create temporary YAML file with alias
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
PRIMARY_DB:
  alias: "DATABASE_URL"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create existing variables
		existingVars := []*model.EnvVar{
			{Name: "DATABASE_URL", Value: "postgres://localhost/myapp", Source: model.SourceDotEnv},
		}

		// Create YAML loader with existing variables
		loader := loader.NewYAMLLoader(yamlPath, existingVars)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "PRIMARY_DB")
		gt.Equal(t, vars[0].Value, "postgres://localhost/myapp")
	})

	t.Run("Cross-reference priority: YAML > .env > system", func(t *testing.T) {
		// Set system environment variable
		os.Setenv("PRIORITY_VAR", "system_value")
		defer os.Unsetenv("PRIORITY_VAR")

		// Create temporary YAML file
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
PRIORITY_VAR:
  value: "yaml_value"

RESULT:
  value: "Value is: {{ .PRIORITY_VAR }}"
  refs: ["PRIORITY_VAR"]
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create existing variables from .env
		existingVars := []*model.EnvVar{
			{Name: "PRIORITY_VAR", Value: "env_value", Source: model.SourceDotEnv},
		}

		// Create YAML loader with existing variables
		loader := loader.NewYAMLLoader(yamlPath, existingVars)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)

		// Find RESULT variable
		var resultVar *model.EnvVar
		for _, v := range vars {
			if v.Name == "RESULT" {
				resultVar = v
				break
			}
		}

		gt.NotNil(t, resultVar)
		// YAML value should take priority
		gt.Equal(t, resultVar.Value, "Value is: yaml_value")
	})

	t.Run("Complex template with mixed sources", func(t *testing.T) {
		// Set system environment variable
		os.Setenv("SYS_HOST", "prod.example.com")
		defer os.Unsetenv("SYS_HOST")

		// Create temporary YAML file
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
APP_NAME:
  value: "myapp"

PORT:
  value: "8080"

FULL_URL:
  value: "https://{{ .ENV_USER }}@{{ .SYS_HOST }}:{{ .PORT }}/{{ .APP_NAME }}"
  refs: ["ENV_USER", "SYS_HOST", "PORT", "APP_NAME"]
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create existing variables from .env
		existingVars := []*model.EnvVar{
			{Name: "ENV_USER", Value: "alice", Source: model.SourceDotEnv},
		}

		// Create YAML loader with existing variables
		loader := loader.NewYAMLLoader(yamlPath, existingVars)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)

		// Find FULL_URL variable
		var fullURLVar *model.EnvVar
		for _, v := range vars {
			if v.Name == "FULL_URL" {
				fullURLVar = v
				break
			}
		}

		gt.NotNil(t, fullURLVar)
		gt.Equal(t, fullURLVar.Value, "https://alice@prod.example.com:8080/myapp")
	})

	t.Run("YAML loader backward compatibility", func(t *testing.T) {
		// Create temporary YAML file
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
SIMPLE_VAR:
  value: "simple_value"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create YAML loader without external variables (backward compatibility)
		loader := loader.NewYAMLLoader(yamlPath)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "SIMPLE_VAR")
		gt.Equal(t, vars[0].Value, "simple_value")
	})

	t.Run("Error on missing variable in template", func(t *testing.T) {
		// Create temporary YAML file with template referencing missing variable
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
TEST_TEMPLATE:
  value: "prefix {{ .MISSING_VAR }} suffix"
  refs: ["MISSING_VAR"]
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create YAML loader
		loader := loader.NewYAMLLoader(yamlPath)

		// Load and resolve - should error
		_, err := loader(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Error on missing variable in alias", func(t *testing.T) {
		// Create temporary YAML file with alias referencing missing variable
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
TEST_ALIAS:
  alias: "MISSING_VAR"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Create YAML loader
		loader := loader.NewYAMLLoader(yamlPath)

		// Load and resolve - should error
		_, err := loader(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Empty string vs missing variable distinction", func(t *testing.T) {
		// Test 1: Empty string variable should work (no error)
		tmpDir := t.TempDir()
		yamlPath1 := filepath.Join(tmpDir, "empty.yaml")
		yamlContent1 := `
TEST_EMPTY:
  value: "prefix {{ .EMPTY_VAR }} suffix"
  refs: ["EMPTY_VAR"]
`
		gt.NoError(t, os.WriteFile(yamlPath1, []byte(yamlContent1), 0644))

		// Create existing variables with empty string
		existingVars := []*model.EnvVar{
			{Name: "EMPTY_VAR", Value: "", Source: model.SourceDotEnv},
		}

		// Create YAML loader with empty variable
		loader1 := loader.NewYAMLLoader(yamlPath1, existingVars)

		// Load and resolve - should succeed with empty string
		vars, err := loader1(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "TEST_EMPTY")
		gt.Equal(t, vars[0].Value, "prefix  suffix") // Empty variable creates empty space

		// Test 2: Missing variable should error
		yamlPath2 := filepath.Join(tmpDir, "missing.yaml")
		yamlContent2 := `
TEST_MISSING:
  value: "prefix {{ .MISSING_VAR }} suffix"
  refs: ["MISSING_VAR"]
`
		gt.NoError(t, os.WriteFile(yamlPath2, []byte(yamlContent2), 0644))

		// Create YAML loader without the required variable
		loader2 := loader.NewYAMLLoader(yamlPath2, existingVars) // MISSING_VAR not in existingVars

		// Load and resolve - should error
		_, err = loader2(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Load YAML file with simple format only", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/simple_format.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 4)

		// Verify values
		expected := map[string]string{
			"DATABASE_URL": "postgres://localhost/mydb",
			"API_KEY":      "secret123",
			"PORT":         "3000",
			"ENV":          "development",
		}

		for _, envVar := range envVars {
			gt.Equal(t, envVar.Source, model.SourceYAML)
			expectedValue, exists := expected[envVar.Name]
			gt.True(t, exists)
			gt.Equal(t, envVar.Value, expectedValue)
		}
	})

	t.Run("Load YAML file with mixed format", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/mixed_format.yaml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// All top-level keys are loaded (no section scope issue in YAML)
		gt.Equal(t, len(envVars), 7)

		// Create a map for easier verification
		envMap := make(map[string]*model.EnvVar)
		for _, env := range envVars {
			envMap[env.Name] = env
		}

		// Verify simple format values
		gt.V(t, envMap["PORT"]).NotNil()
		gt.Equal(t, envMap["PORT"].Value, "3000")
		gt.V(t, envMap["ENV"]).NotNil()
		gt.Equal(t, envMap["ENV"].Value, "development")
		gt.V(t, envMap["DEBUG"]).NotNil()
		gt.Equal(t, envMap["DEBUG"].Value, "true")

		// Verify structured format values
		gt.V(t, envMap["DATABASE_URL"]).NotNil()
		gt.Equal(t, envMap["DATABASE_URL"].Value, "postgres://localhost/mydb")
		gt.V(t, envMap["SSL_CERT"]).NotNil()
		gt.Equal(t, envMap["SSL_CERT"].Value, "config file content")
		gt.V(t, envMap["LOG_LEVEL"]).NotNil()
		gt.Equal(t, envMap["LOG_LEVEL"].Value, "info")

		// TOKEN is a valid top-level key in YAML (no section scope issue)
		gt.V(t, envMap["TOKEN"]).NotNil()
		gt.Equal(t, envMap["TOKEN"].Value, "hoge")

		// All should have YAML source
		for _, envVar := range envVars {
			gt.Equal(t, envVar.Source, model.SourceYAML)
		}
	})

	t.Run("Numbers in simple format are converted to strings", func(t *testing.T) {
		// YAML numbers are automatically converted to strings
		tmpFile := filepath.Join(t.TempDir(), "number_value.yaml")
		err := os.WriteFile(tmpFile, []byte("PORT: 123\nDEBUG: true\nNAME: \"test\""), 0644)
		gt.NoError(t, err)

		loadFunc := loader.NewYAMLLoader(tmpFile)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(envVars), 3)

		envMap := make(map[string]*model.EnvVar)
		for _, env := range envVars {
			envMap[env.Name] = env
		}

		// Numbers and booleans are converted to strings
		gt.Equal(t, envMap["PORT"].Value, "123")
		gt.Equal(t, envMap["DEBUG"].Value, "true")
		gt.Equal(t, envMap["NAME"].Value, "test")
	})

	t.Run("Command with refs", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/command_with_refs.yaml")
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		testCases := []struct {
			name        string
			varName     string
			expectedVal string
		}{
			{
				name:        "single ref",
				varName:     "SIMPLE_MESSAGE",
				expectedVal: "Hello World",
			},
			{
				name:        "multiple refs",
				varName:     "MESSAGE",
				expectedVal: "Server: prod-server:8080",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				gt.Equal(t, varMap[tc.varName], tc.expectedVal)
			})
		}

		// Verify the referenced variables are present
		gt.Equal(t, varMap["NAME"], "World")
		gt.Equal(t, varMap["HOST"], "prod-server")
		gt.Equal(t, varMap["PORT"], "8080")
	})

	t.Run("Command with circular refs should fail", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/command_circular_refs.yaml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Command without refs should work", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoader("testdata/with_command.yaml")
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// HOSTNAME should execute ["echo", "test-host"]
		gt.Equal(t, varMap["HOSTNAME"], "test-host")
	})
}

func TestProfileBasic(t *testing.T) {
	t.Run("loads default values when no profile specified", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_basic.yaml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
		gt.Equal(t, varMap["DB_HOST"], "localhost")
		gt.Equal(t, varMap["LOG_LEVEL"], "info")
	})

	t.Run("loads dev profile values", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_basic.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_URL"], "http://localhost:8080")
		gt.Equal(t, varMap["DB_HOST"], "localhost")
		gt.Equal(t, varMap["LOG_LEVEL"], "debug")
	})

	t.Run("loads staging profile values", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_basic.yaml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_URL"], "https://staging-api.example.com")
		gt.Equal(t, varMap["DB_HOST"], "staging-db.example.com")
		gt.Equal(t, varMap["LOG_LEVEL"], "info")
	})

	t.Run("loads prod profile values", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_basic.yaml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
		gt.Equal(t, varMap["DB_HOST"], "prod-db.example.com")
		gt.Equal(t, varMap["LOG_LEVEL"], "error")
	})

	t.Run("unknown profile uses default values", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_basic.yaml", "unknown", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
		gt.Equal(t, varMap["DB_HOST"], "localhost")
		gt.Equal(t, varMap["LOG_LEVEL"], "info")
	})
}

func TestProfileUnset(t *testing.T) {
	t.Run("empty object unsets variable in prod profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_unset.yaml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// DEBUG_FLAG and VERBOSE_MODE should not exist
		_, hasDebugFlag := varMap["DEBUG_FLAG"]
		gt.False(t, hasDebugFlag)

		_, hasVerboseMode := varMap["VERBOSE_MODE"]
		gt.False(t, hasVerboseMode)

		// Other variables should exist
		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
	})

	t.Run("dev profile has all variables", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_unset.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DEBUG_FLAG"], "true")
		gt.Equal(t, varMap["VERBOSE_MODE"], "enabled")
		gt.Equal(t, varMap["API_URL"], "http://localhost:8080")
	})

	t.Run("default profile has all variables", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_unset.yaml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DEBUG_FLAG"], "false")
		gt.Equal(t, varMap["VERBOSE_MODE"], "disabled")
		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
	})
}

func TestProfileAdvanced(t *testing.T) {
	t.Run("profile with file reference", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_advanced.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// SSL_CERT uses value in dev profile
		gt.S(t, varMap["SSL_CERT"]).Contains("BEGIN CERTIFICATE")
	})

	t.Run("profile with value override", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_advanced.yaml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// ENV has default value
		gt.Equal(t, varMap["ENV"], "default")
	})

	t.Run("profile with command execution", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_advanced.yaml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["HOSTNAME"], "prod-server")
	})

	t.Run("profile with alias", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_advanced.yaml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// API_KEY uses alias to PRIMARY_DB (no staging profile)
		gt.Equal(t, varMap["API_KEY"], "postgres://localhost/primary")
	})

	t.Run("profile with template", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_advanced.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// HOSTNAME uses default command in dev (returns "default-host")
		gt.Equal(t, varMap["HOSTNAME"], "dev-server")
		// CONFIG_PATH uses dev profile
		gt.Equal(t, varMap["CONFIG_PATH"], "/home/dev/config.yml")
	})
}

func TestProfileCompatibility(t *testing.T) {
	t.Run("backward compatibility with no profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_compat.yaml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// Variables without profiles should work normally
		gt.Equal(t, varMap["SIMPLE_VAR"], "simple_value")
		gt.Equal(t, varMap["COMPLEX_VAR"], "complex_value")

		// Variables with profiles should use default values
		gt.Equal(t, varMap["PROFILE_VAR"], "default_profile_value")
	})

	t.Run("profile selection with mixed variables", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_compat.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// Non-profile variables should remain unchanged
		gt.Equal(t, varMap["SIMPLE_VAR"], "simple_value")
		gt.Equal(t, varMap["COMPLEX_VAR"], "complex_value")

		// Profile variable should use dev value
		gt.Equal(t, varMap["PROFILE_VAR"], "dev_profile_value")
		// MIXED_VAR has no dev profile, uses default
		gt.Equal(t, varMap["MIXED_VAR"], "default_mixed")
	})

	t.Run("profile with partial coverage", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_compat.yaml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// MIXED_VAR has staging profile
		gt.Equal(t, varMap["MIXED_VAR"], "staging_mixed")
		// PROFILE_VAR has no staging profile, uses default
		gt.Equal(t, varMap["PROFILE_VAR"], "default_profile_value")
	})
}

func TestProfileNestedValidation(t *testing.T) {
	t.Run("nested profile in YAML should be rejected", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_nested.yaml", "", nil)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("nested profile")
	})
}

func TestProfileDottedKeyFormat(t *testing.T) {
	t.Run("self-referencing dotted key format", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_self_reference.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_TOKEN"], "dev-token")
	})

	t.Run("self-referencing dotted key default", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_self_reference.yaml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_TOKEN"], "default-token")
	})

	t.Run("dotted key format with dev profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_dotted.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "dev@example.com")
		gt.Equal(t, varMap["API_KEY"], "dev-key-123")
	})

	t.Run("dotted key format with staging profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_dotted.yaml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "staging@example.com")
		gt.Equal(t, varMap["API_KEY"], "default-key")
	})

	t.Run("dotted key format with prod profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_dotted.yaml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "prod@example.com")
		gt.Equal(t, varMap["API_KEY"], "prod-key-456")
	})

	t.Run("dotted key format with default profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_dotted.yaml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "default@example.com")
		gt.Equal(t, varMap["API_KEY"], "default-key")
	})
}

func TestProfileInlineFormat(t *testing.T) {
	t.Run("inline table format should work", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_inline.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DB_HOST"], "localhost")
		gt.Equal(t, varMap["API_URL"], "http://localhost:8080")
	})

	t.Run("inline format with staging profile", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_inline.yaml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DB_HOST"], "staging-db.example.com")
		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
	})

	t.Run("inline string format should work", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_inline_string.yaml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "dev@example.com")
		gt.Equal(t, varMap["API_KEY"], "dev-key-123")
	})

	t.Run("inline string format default values", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_inline_string.yaml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "default@example.com")
		gt.Equal(t, varMap["API_KEY"], "default-key")
	})
}

func TestProfileNilHandling(t *testing.T) {
	t.Run("nil profile value should be skipped", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_unset.yaml", "nonexistent", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DEBUG_FLAG"], "false")
		gt.Equal(t, varMap["VERBOSE_MODE"], "disabled")
		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
	})

	t.Run("empty object unsets variable correctly", func(t *testing.T) {
		loadFunc := loader.NewYAMLLoaderWithProfile("testdata/profile_unset.yaml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		_, hasDebugFlag := varMap["DEBUG_FLAG"]
		_, hasVerboseMode := varMap["VERBOSE_MODE"]
		gt.False(t, hasDebugFlag)
		gt.False(t, hasVerboseMode)

		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
	})
}

func TestYAMLFileExtensions(t *testing.T) {
	t.Run("Load .env.yml file only", func(t *testing.T) {
		tmpDir := t.TempDir()
		ymlPath := filepath.Join(tmpDir, ".env.yml")
		ymlContent := `
DATABASE_URL: "postgres://localhost/testdb"
API_KEY: "secret123"
`
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(filepath.Join(tmpDir, ".env.yaml"))
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 2)
		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DATABASE_URL"], "postgres://localhost/testdb")
		gt.Equal(t, varMap["API_KEY"], "secret123")
	})

	t.Run("Load .env.yaml file only", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		yamlContent := `
DATABASE_URL: "postgres://localhost/yamldb"
API_KEY: "yaml-secret"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 2)
		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DATABASE_URL"], "postgres://localhost/yamldb")
		gt.Equal(t, varMap["API_KEY"], "yaml-secret")
	})

	t.Run("Merge both files with different keys", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
DATABASE_URL: "postgres://localhost/db"
API_KEY: "key123"
`
		ymlContent := `
REDIS_URL: "redis://localhost:6379"
LOG_LEVEL: "info"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 4)
		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DATABASE_URL"], "postgres://localhost/db")
		gt.Equal(t, varMap["API_KEY"], "key123")
		gt.Equal(t, varMap["REDIS_URL"], "redis://localhost:6379")
		gt.Equal(t, varMap["LOG_LEVEL"], "info")
	})

	t.Run("Merge both files: value+refs already in .yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		// Simple case: .yaml has complete config, .yml has other vars
		yamlContent := `
DB_USER: "admin"
DB_PASS: "secret"
DB_URL:
  value: "postgres://{{ .DB_USER }}:{{ .DB_PASS }}@localhost/db"
  refs: ["DB_USER", "DB_PASS"]
`
		ymlContent := `
REDIS_URL: "redis://localhost:6379"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DB_URL"], "postgres://admin:secret@localhost/db")
	})

	t.Run("Merge both files: refs merging from both", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
VAR1: "value1"
VAR2: "value2"
VAR3: "value3"
TEMPLATE:
  value: "{{ .VAR1 }} {{ .VAR2 }} {{ .VAR3 }}"
  refs: ["VAR1", "VAR2"]
`
		ymlContent := `
TEMPLATE:
  refs: ["VAR2", "VAR3"]
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// Should merge refs and execute template correctly
		gt.Equal(t, varMap["TEMPLATE"], "value1 value2 value3")
	})

	t.Run("Merge both files: profile merging", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
API_URL:
  value: "https://api.example.com"
  profile:
    dev:
      value: "http://localhost:8080"
`
		ymlContent := `
API_URL:
  profile:
    staging:
      value: "https://staging.example.com"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		// Test dev profile
		loadFuncDev := loader.NewYAMLLoaderWithProfile(yamlPath, "dev", nil)
		envVarsDev := gt.R1(loadFuncDev(context.Background())).NoError(t)
		gt.Equal(t, envVarsDev[0].Value, "http://localhost:8080")

		// Test staging profile
		loadFuncStaging := loader.NewYAMLLoaderWithProfile(yamlPath, "staging", nil)
		envVarsStaging := gt.R1(loadFuncStaging(context.Background())).NoError(t)
		gt.Equal(t, envVarsStaging[0].Value, "https://staging.example.com")

		// Test default profile
		loadFuncDefault := loader.NewYAMLLoaderWithProfile(yamlPath, "", nil)
		envVarsDefault := gt.R1(loadFuncDefault(context.Background())).NoError(t)
		gt.Equal(t, envVarsDefault[0].Value, "https://api.example.com")
	})

	t.Run("Merge both files: .yml profile overrides .yaml profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
API_URL:
  value: "https://api.example.com"
  profile:
    dev:
      value: "http://localhost:8080"
`
		ymlContent := `
API_URL:
  profile:
    dev:
      value: "http://localhost:9090"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoaderWithProfile(yamlPath, "dev", nil)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		gt.Equal(t, envVars[0].Value, "http://localhost:9090")
	})

	t.Run("Error: conflicting value fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
DATABASE_URL:
  value: "postgres://localhost/db1"
`
		ymlContent := `
DATABASE_URL:
  value: "postgres://localhost/db2"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("conflicting field \"value\"")
		gt.S(t, err.Error()).Contains("DATABASE_URL")
	})

	t.Run("Error: conflicting file fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
SSL_CERT:
  file: "/path/to/cert1.pem"
`
		ymlContent := `
SSL_CERT:
  file: "/path/to/cert2.pem"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("conflicting field \"file\"")
		gt.S(t, err.Error()).Contains("SSL_CERT")
	})

	t.Run("Error: conflicting command fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
HOSTNAME:
  command: ["echo", "host1"]
`
		ymlContent := `
HOSTNAME:
  command: ["echo", "host2"]
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("conflicting field \"command\"")
		gt.S(t, err.Error()).Contains("HOSTNAME")
	})

	t.Run("Error: conflicting alias fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		ymlPath := filepath.Join(tmpDir, ".env.yml")

		yamlContent := `
PRIMARY_DB: "postgres://localhost/primary"
DB_URL:
  alias: "PRIMARY_DB"
`
		ymlContent := `
SECONDARY_DB: "postgres://localhost/secondary"
DB_URL:
  alias: "SECONDARY_DB"
`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		loadFunc := loader.NewYAMLLoader(yamlPath)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("conflicting field \"alias\"")
		gt.S(t, err.Error()).Contains("DB_URL")
	})

	t.Run("No files exist returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")

		loadFunc := loader.NewYAMLLoader(yamlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		gt.Nil(t, envVars)
	})

	t.Run("Load .yml file by passing .yml path directly", func(t *testing.T) {
		tmpDir := t.TempDir()
		ymlPath := filepath.Join(tmpDir, ".env.yml")
		ymlContent := `DATABASE_URL: "postgres://localhost/testdb"
API_KEY: "secret123"`
		gt.NoError(t, os.WriteFile(ymlPath, []byte(ymlContent), 0644))

		// Pass .yml path directly - should not try to load the same file twice
		loadFunc := loader.NewYAMLLoader(ymlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 2)
		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DATABASE_URL"], "postgres://localhost/testdb")
		gt.Equal(t, varMap["API_KEY"], "secret123")
	})

	t.Run("Load .yaml file by passing .yaml path directly", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, ".env.yaml")
		yamlContent := `DATABASE_URL: "postgres://localhost/yamldb"
PORT: "8080"`
		gt.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

		// Pass .yaml path directly
		loadFunc := loader.NewYAMLLoader(yamlPath)
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 2)
		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DATABASE_URL"], "postgres://localhost/yamldb")
		gt.Equal(t, varMap["PORT"], "8080")
	})
}
