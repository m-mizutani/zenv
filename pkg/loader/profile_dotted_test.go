package loader_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
)

func TestProfileDottedKeyFormat(t *testing.T) {
	t.Run("self-referencing dotted key format", func(t *testing.T) {
		// Test the format: KEY.profile.dev inside [KEY] section
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_self_reference.toml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_TOKEN"], "dev-token")
	})

	t.Run("self-referencing dotted key default", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_self_reference.toml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["API_TOKEN"], "default-token")
	})

	t.Run("dotted key format with dev profile", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_dotted.toml", "dev", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_dotted.toml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["EMAIL"], "staging@example.com")
		// API_KEY should use default (no staging profile)
		gt.Equal(t, varMap["API_KEY"], "default-key")
	})

	t.Run("dotted key format with prod profile", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_dotted.toml", "prod", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_dotted.toml", "", nil)
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
