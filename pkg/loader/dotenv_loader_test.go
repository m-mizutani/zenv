package loader_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestDotEnvLoader(t *testing.T) {

	t.Run("Load valid .env file", func(t *testing.T) {
		// Create temporary .env file
		tmpFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `# Comment line
KEY1=value1
KEY2="quoted value"
KEY3='single quoted'

# Another comment
KEY4=value with spaces`

		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Test loader
		loadFunc := loader.NewDotEnvLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)

		gt.Equal(t, len(envVars), 4)

		// Verify values
		expected := map[string]string{
			"KEY1": "value1",
			"KEY2": "quoted value",
			"KEY3": "single quoted",
			"KEY4": "value with spaces",
		}

		for _, envVar := range envVars {
			gt.Equal(t, envVar.Source, model.SourceDotEnv)
			expectedValue, exists := expected[envVar.Name]
			gt.True(t, exists)
			gt.Equal(t, envVar.Value, expectedValue)
		}
	})

	t.Run("Handle non-existent file", func(t *testing.T) {
		loadFunc := loader.NewDotEnvLoader("non_existent.env")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		gt.Nil(t, envVars)
	})

	t.Run("Handle invalid format", func(t *testing.T) {
		// Create temporary .env file with invalid format
		tmpFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `INVALID_LINE_WITHOUT_EQUALS
VALID_KEY=valid_value`

		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Test loader
		loadFunc := loader.NewDotEnvLoader(tmpFile.Name())
		_, err := loadFunc(context.Background())
		
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid format")
	})

	t.Run("Handle empty lines and comments", func(t *testing.T) {
		// Create temporary .env file
		tmpFile := gt.R1(os.CreateTemp("", "test*.env")).NoError(t)
		defer os.Remove(tmpFile.Name())

		content := `
# This is a comment
KEY1=value1

# Another comment

KEY2=value2
`

		gt.R1(tmpFile.WriteString(content)).NoError(t)
		tmpFile.Close()

		// Test loader
		loadFunc := loader.NewDotEnvLoader(tmpFile.Name())
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		
		gt.Equal(t, len(envVars), 2)
	})
}
