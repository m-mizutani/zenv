package loader_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
)

func TestProfileInlineFormat(t *testing.T) {
	t.Run("inline table format should work", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_inline.toml", "dev", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_inline.toml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DB_HOST"], "staging-db.example.com")
		// API_URL should use default (no staging profile)
		gt.Equal(t, varMap["API_URL"], "https://api.example.com")
	})

	t.Run("inline string format should work", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_inline_string.toml", "dev", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_inline_string.toml", "", nil)
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
