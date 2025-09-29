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

func TestTOMLLoader(t *testing.T) {

	t.Run("Load valid TOML file with static values", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/valid.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 3)

		// Verify values
		expected := map[string]string{
			"DATABASE_URL":     "postgresql://localhost/testdb",
			"API_KEY":          "secret123",
			"MULTILINE_CONFIG": "line1\nline2\nline3",
		}

		for _, envVar := range envVars {
			gt.Equal(t, envVar.Source, model.SourceTOML)
			expectedValue, exists := expected[envVar.Name]
			gt.True(t, exists)
			gt.Equal(t, envVar.Value, expectedValue)
		}
	})

	t.Run("Load TOML file with file reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/with_file.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "CONFIG_DATA")
		gt.Equal(t, envVars[0].Value, "config file content")
		gt.Equal(t, envVars[0].Source, model.SourceTOML)
	})

	t.Run("Load TOML file with command execution", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/with_command.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "HOSTNAME")
		gt.Equal(t, envVars[0].Value, "test-host")
		gt.Equal(t, envVars[0].Source, model.SourceTOML)
	})

	t.Run("Handle non-existent file", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/non_existent.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		gt.Nil(t, envVars)
	})

	t.Run("Handle invalid TOML syntax", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/invalid_syntax.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to parse TOML file")
	})

	t.Run("Handle validation errors", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/validation_error.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	t.Run("Handle missing file reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/missing_file.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to read file")
	})

	t.Run("Handle command execution failure", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/command_failure.toml")
		_, err := loadFunc(context.Background())
		// The 'false' command exits with non-zero status
		gt.Error(t, err)
	})

	t.Run("Load TOML file with alias", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/alias.toml")
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

		loadFunc := loader.NewTOMLLoader("testdata/alias_system.toml")
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

	t.Run("Alias precedence: TOML overrides system environment", func(t *testing.T) {
		// Set a system environment variable
		t.Setenv("SHARED_VAR", "system_value")

		// Create a TOML file that defines the same variable and an alias to it
		tomlContent := `
[SHARED_VAR]
value = "toml_value"

[ALIAS_TO_SHARED]
alias = "SHARED_VAR"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_precedence*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// Find the ALIAS_TO_SHARED variable
		var aliasValue string
		for _, envVar := range envVars {
			if envVar.Name == "ALIAS_TO_SHARED" {
				aliasValue = envVar.Value
				break
			}
		}

		// The alias should resolve to the TOML value, not the system value
		gt.Equal(t, aliasValue, "toml_value")
	})

	t.Run("Alias resolves empty system environment variable", func(t *testing.T) {
		// Set an empty system environment variable
		t.Setenv("EMPTY_SYSTEM_VAR", "")

		tomlContent := `
[ALIAS_TO_EMPTY]
alias = "EMPTY_SYSTEM_VAR"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_empty*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
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
		loadFunc := loader.NewTOMLLoader("testdata/alias_missing.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Handle circular alias reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/circular_alias.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Validation error when multiple types including alias", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/alias_multiple_types.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	// Template tests
	t.Run("Load TOML file with basic template", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/template.toml")
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
		loadFunc := loader.NewTOMLLoader("testdata/template.toml")
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

		// Create a TOML file that uses system var in template
		tomlContent := `
[FROM_SYSTEM]
template = "System: {{ .TEST_SYS_VAR }}"
refs = ["TEST_SYS_VAR"]
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_sys*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
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
		loadFunc := loader.NewTOMLLoader("testdata/template_complex.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// WITH_EMPTY should have empty brackets
		gt.Equal(t, vars["WITH_EMPTY"], "Value: []")

		// MISSING_REF_TEMPLATE should not exist anymore (removed from template_complex.toml)
		_, exists := vars["MISSING_REF_TEMPLATE"]
		gt.False(t, exists)
	})

	t.Run("Template with nested references", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/template_complex.toml")
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
		loadFunc := loader.NewTOMLLoader("testdata/template_complex.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		vars := make(map[string]string)
		for _, envVar := range envVars {
			vars[envVar.Name] = envVar.Value
		}

		// LOG_CONFIG should be "debug" when DEBUG_MODE is true
		gt.Equal(t, vars["LOG_CONFIG"], "debug")
	})

	t.Run("Handle circular template reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/template_circular.toml")
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Template syntax error", func(t *testing.T) {
		// Create a TOML file with template syntax error
		tomlContent := `
[SYNTAX_ERROR]
template = "{{ .VAR_NAME"
refs = ["VAR_NAME"]

[VAR_NAME]
value = "test"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_syntax*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to parse template")
	})

	t.Run("Template without refs field", func(t *testing.T) {
		// Create a TOML file with template but no refs (should work)
		tomlContent := `
[NO_REFS]
template = "Static template text"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_norefs*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "NO_REFS")
		gt.Equal(t, envVars[0].Value, "Static template text")
	})

	t.Run("Refs without template field", func(t *testing.T) {
		// Create a TOML file with refs but no template
		tomlContent := `
[REFS_ONLY]
refs = ["SOME_VAR"]
`
		tmpFile := gt.R1(os.CreateTemp("", "test_refs_only*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("refs can only be used with template")
	})

	t.Run("Template with multiple value types", func(t *testing.T) {
		// Create a TOML file with both value and template
		tomlContent := `
[MULTIPLE_TYPES]
value = "value"
template = "{{ .OTHER }}"
refs = ["OTHER"]
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_multiple*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("multiple value types specified")
	})

	t.Run("Template referenced by an alias", func(t *testing.T) {
		tomlContent := `
[TEMPLATE_VAR]
template = "hello world"
refs = []

[ALIAS_VAR]
alias = "TEMPLATE_VAR"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_template_alias*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
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
		loadFunc := loader.NewTOMLLoader("testdata/alias_to_template.toml")
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
		loadFunc := loader.NewTOMLLoader("testdata/complex_circular.toml")
		_, err := loadFunc(context.Background())

		// Should detect circular reference in any of the complex cases
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Circular reference via alias and template mix", func(t *testing.T) {
		// Create a specific test case: A->B(template)->C->A
		tomlContent := `
[VAR_A]
alias = "VAR_B"

[VAR_B]
template = "B uses {{ .VAR_C }}"
refs = ["VAR_C"]

[VAR_C]
alias = "VAR_A"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_circular_mix*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	t.Run("Template self-reference through alias", func(t *testing.T) {
		// Create a test where template references itself through an alias
		tomlContent := `
[SELF_TEMPLATE]
template = "Self: {{ .SELF_ALIAS }}"
refs = ["SELF_ALIAS"]

[SELF_ALIAS]
alias = "SELF_TEMPLATE"
`
		tmpFile := gt.R1(os.CreateTemp("", "test_self_ref_mix*.toml")).NoError(t)
		defer os.Remove(tmpFile.Name())
		gt.R1(tmpFile.WriteString(tomlContent)).NoError(t)
		tmpFile.Close()

		loadFunc := loader.NewTOMLLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())

		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("circular reference")
	})

	// Cross-reference tests: TOML can reference .env and system variables
	t.Run("Template can reference .env variables", func(t *testing.T) {
		// Create temporary TOML file with template
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[DB_URL]
template = "postgres://{{ .DB_USER }}:{{ .DB_PASS }}@{{ .DB_HOST }}:5432/mydb"
refs = ["DB_USER", "DB_PASS", "DB_HOST"]
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create existing variables (as if from .env file)
		existingVars := []*model.EnvVar{
			{Name: "DB_USER", Value: "admin", Source: model.SourceDotEnv},
			{Name: "DB_PASS", Value: "secret123", Source: model.SourceDotEnv},
			{Name: "DB_HOST", Value: "localhost", Source: model.SourceDotEnv},
		}

		// Create TOML loader with existing variables
		loader := loader.NewTOMLLoader(tomlPath, existingVars)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "DB_URL")
		gt.Equal(t, vars[0].Value, "postgres://admin:secret123@localhost:5432/mydb")
	})

	t.Run("Alias can reference .env variables", func(t *testing.T) {
		// Create temporary TOML file with alias
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[PRIMARY_DB]
alias = "DATABASE_URL"
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create existing variables
		existingVars := []*model.EnvVar{
			{Name: "DATABASE_URL", Value: "postgres://localhost/myapp", Source: model.SourceDotEnv},
		}

		// Create TOML loader with existing variables
		loader := loader.NewTOMLLoader(tomlPath, existingVars)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "PRIMARY_DB")
		gt.Equal(t, vars[0].Value, "postgres://localhost/myapp")
	})

	t.Run("Cross-reference priority: TOML > .env > system", func(t *testing.T) {
		// Set system environment variable
		os.Setenv("PRIORITY_VAR", "system_value")
		defer os.Unsetenv("PRIORITY_VAR")

		// Create temporary TOML file
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[PRIORITY_VAR]
value = "toml_value"

[RESULT]
template = "Value is: {{ .PRIORITY_VAR }}"
refs = ["PRIORITY_VAR"]
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create existing variables from .env
		existingVars := []*model.EnvVar{
			{Name: "PRIORITY_VAR", Value: "env_value", Source: model.SourceDotEnv},
		}

		// Create TOML loader with existing variables
		loader := loader.NewTOMLLoader(tomlPath, existingVars)

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
		// TOML value should take priority
		gt.Equal(t, resultVar.Value, "Value is: toml_value")
	})

	t.Run("Complex template with mixed sources", func(t *testing.T) {
		// Set system environment variable
		os.Setenv("SYS_HOST", "prod.example.com")
		defer os.Unsetenv("SYS_HOST")

		// Create temporary TOML file
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[APP_NAME]
value = "myapp"

[PORT]
value = "8080"

[FULL_URL]
template = "https://{{ .ENV_USER }}@{{ .SYS_HOST }}:{{ .PORT }}/{{ .APP_NAME }}"
refs = ["ENV_USER", "SYS_HOST", "PORT", "APP_NAME"]
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create existing variables from .env
		existingVars := []*model.EnvVar{
			{Name: "ENV_USER", Value: "alice", Source: model.SourceDotEnv},
		}

		// Create TOML loader with existing variables
		loader := loader.NewTOMLLoader(tomlPath, existingVars)

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

	t.Run("TOML loader backward compatibility", func(t *testing.T) {
		// Create temporary TOML file
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[SIMPLE_VAR]
value = "simple_value"
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create TOML loader without external variables (backward compatibility)
		loader := loader.NewTOMLLoader(tomlPath)

		// Load and resolve
		vars, err := loader(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "SIMPLE_VAR")
		gt.Equal(t, vars[0].Value, "simple_value")
	})

	t.Run("Error on missing variable in template", func(t *testing.T) {
		// Create temporary TOML file with template referencing missing variable
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[TEST_TEMPLATE]
template = "prefix {{ .MISSING_VAR }} suffix"
refs = ["MISSING_VAR"]
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create TOML loader
		loader := loader.NewTOMLLoader(tomlPath)

		// Load and resolve - should error
		_, err := loader(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Error on missing variable in alias", func(t *testing.T) {
		// Create temporary TOML file with alias referencing missing variable
		tmpDir := t.TempDir()
		tomlPath := filepath.Join(tmpDir, "test.toml")
		tomlContent := `
[TEST_ALIAS]
alias = "MISSING_VAR"
`
		gt.NoError(t, os.WriteFile(tomlPath, []byte(tomlContent), 0644))

		// Create TOML loader
		loader := loader.NewTOMLLoader(tomlPath)

		// Load and resolve - should error
		_, err := loader(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Empty string vs missing variable distinction", func(t *testing.T) {
		// Test 1: Empty string variable should work (no error)
		tmpDir := t.TempDir()
		tomlPath1 := filepath.Join(tmpDir, "empty.toml")
		tomlContent1 := `
[TEST_EMPTY]
template = "prefix {{ .EMPTY_VAR }} suffix"
refs = ["EMPTY_VAR"]
`
		gt.NoError(t, os.WriteFile(tomlPath1, []byte(tomlContent1), 0644))

		// Create existing variables with empty string
		existingVars := []*model.EnvVar{
			{Name: "EMPTY_VAR", Value: "", Source: model.SourceDotEnv},
		}

		// Create TOML loader with empty variable
		loader1 := loader.NewTOMLLoader(tomlPath1, existingVars)

		// Load and resolve - should succeed with empty string
		vars, err := loader1(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, len(vars), 1)
		gt.Equal(t, vars[0].Name, "TEST_EMPTY")
		gt.Equal(t, vars[0].Value, "prefix  suffix") // Empty variable creates empty space

		// Test 2: Missing variable should error
		tomlPath2 := filepath.Join(tmpDir, "missing.toml")
		tomlContent2 := `
[TEST_MISSING]
template = "prefix {{ .MISSING_VAR }} suffix"
refs = ["MISSING_VAR"]
`
		gt.NoError(t, os.WriteFile(tomlPath2, []byte(tomlContent2), 0644))

		// Create TOML loader without the required variable
		loader2 := loader.NewTOMLLoader(tomlPath2, existingVars) // MISSING_VAR not in existingVars

		// Load and resolve - should error
		_, err = loader2(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("variable not found")
	})

	t.Run("Load TOML file with simple format only", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/simple_format.toml")
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
			gt.Equal(t, envVar.Source, model.SourceTOML)
			expectedValue, exists := expected[envVar.Name]
			gt.True(t, exists)
			gt.Equal(t, envVar.Value, expectedValue)
		}
	})

	t.Run("Load TOML file with mixed format", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoader("testdata/mixed_format.toml")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		// TOKEN after section becomes part of LOG_LEVEL section, so only 6 top-level entries
		gt.Equal(t, len(envVars), 6)

		// Create a map for easier verification
		envMap := make(map[string]*model.EnvVar)
		for _, env := range envVars {
			envMap[env.Name] = env
		}

		// Verify simple format values (before sections)
		gt.V(t, envMap["PORT"]).NotNil()
		gt.Equal(t, envMap["PORT"].Value, "3000")
		gt.V(t, envMap["ENV"]).NotNil()
		gt.Equal(t, envMap["ENV"].Value, "development")
		gt.V(t, envMap["DEBUG"]).NotNil()
		gt.Equal(t, envMap["DEBUG"].Value, "true")

		// Verify section format values
		gt.V(t, envMap["DATABASE_URL"]).NotNil()
		gt.Equal(t, envMap["DATABASE_URL"].Value, "postgres://localhost/mydb")
		gt.V(t, envMap["SSL_CERT"]).NotNil()
		gt.Equal(t, envMap["SSL_CERT"].Value, "config file content")
		gt.V(t, envMap["LOG_LEVEL"]).NotNil()
		gt.Equal(t, envMap["LOG_LEVEL"].Value, "info")

		// TOKEN after section is NOT a top-level key (becomes part of LOG_LEVEL section)
		gt.V(t, envMap["TOKEN"]).Nil()

		// All should have TOML source
		for _, envVar := range envVars {
			gt.Equal(t, envVar.Source, model.SourceTOML)
		}
	})

	t.Run("Invalid type in simple format should fail", func(t *testing.T) {
		// Create a temporary file with invalid content
		tmpFile := filepath.Join(t.TempDir(), "invalid_simple.toml")
		err := os.WriteFile(tmpFile, []byte("INVALID_NUMBER = 123\nVALID_STRING = \"test\""), 0644)
		gt.NoError(t, err)

		loadFunc := loader.NewTOMLLoader(tmpFile)
		_, err = loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("unsupported type")
	})
}
