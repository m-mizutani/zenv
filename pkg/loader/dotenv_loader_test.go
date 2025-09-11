package loader_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestDotEnvLoader(t *testing.T) {

	t.Run("Load valid .env file", func(t *testing.T) {
		// Create temporary .env file
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `# Comment line
KEY1=value1
KEY2="quoted value"
KEY3='single quoted'

# Another comment
KEY4=value with spaces`

		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Test loader
		loadFunc := loader.NewDotEnvLoader(tmpFile.Name())
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if len(envVars) != 4 {
			t.Errorf("expected 4 env vars, got %d", len(envVars))
		}

		// Verify values
		expected := map[string]string{
			"KEY1": "value1",
			"KEY2": "quoted value",
			"KEY3": "single quoted",
			"KEY4": "value with spaces",
		}

		for _, envVar := range envVars {
			if envVar.Source != model.SourceDotEnv {
				t.Errorf("expected source %v, got %v", model.SourceDotEnv, envVar.Source)
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

	t.Run("Handle non-existent file", func(t *testing.T) {
		loadFunc := loader.NewDotEnvLoader("non_existent.env")
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if envVars != nil {
			t.Errorf("expected nil envVars, got %v", envVars)
		}
	})

	t.Run("Handle invalid format", func(t *testing.T) {
		// Create temporary .env file with invalid format
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `INVALID_LINE_WITHOUT_EQUALS
VALID_KEY=valid_value`

		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Test loader
		loadFunc := loader.NewDotEnvLoader(tmpFile.Name())
		_, err = loadFunc(context.Background())

		if err == nil {
			t.Error("expected error for invalid format")
		}
		if !strings.Contains(err.Error(), "invalid format") {
			t.Errorf("expected error to contain 'invalid format', got %s", err.Error())
		}
	})

	t.Run("Handle empty lines and comments", func(t *testing.T) {
		// Create temporary .env file
		tmpFile, err := os.CreateTemp("", "test*.env")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		content := `
# This is a comment
KEY1=value1

# Another comment

KEY2=value2
`

		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}
		tmpFile.Close()

		// Test loader
		loadFunc := loader.NewDotEnvLoader(tmpFile.Name())
		envVars, err := loadFunc(context.Background())

		if err != nil {
			t.Fatal(err)
		}
		if len(envVars) != 2 {
			t.Errorf("expected 2 env vars, got %d", len(envVars))
		}
	})
}
