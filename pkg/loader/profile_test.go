package loader_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
)

func TestProfileBasic(t *testing.T) {
	t.Run("loads default values when no profile specified", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_basic.toml", "", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_basic.toml", "dev", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_basic.toml", "staging", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_basic.toml", "prod", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_basic.toml", "unknown", nil)
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
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_unset.toml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// DEBUG_FLAG and VERBOSE_MODE should not exist
		_, hasDebugFlag := varMap["DEBUG_FLAG"]
		_, hasVerboseMode := varMap["VERBOSE_MODE"]
		gt.False(t, hasDebugFlag)
		gt.False(t, hasVerboseMode)

		// FEATURE_FLAG should still exist (no profile for prod)
		gt.Equal(t, varMap["FEATURE_FLAG"], "on")
	})

	t.Run("dev profile has all variables", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_unset.toml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DEBUG_FLAG"], "true")
		gt.Equal(t, varMap["VERBOSE_MODE"], "enabled")
		gt.Equal(t, varMap["FEATURE_FLAG"], "on")
	})

	t.Run("default profile has all variables", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_unset.toml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		gt.Equal(t, varMap["DEBUG_FLAG"], "true")
		gt.Equal(t, varMap["VERBOSE_MODE"], "enabled")
		gt.Equal(t, varMap["FEATURE_FLAG"], "on")
	})
}

func TestProfileAdvanced(t *testing.T) {
	t.Run("profile with file reference", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_advanced.toml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// SSL_CERT should load from file in prod
		gt.Equal(t, varMap["SSL_CERT"], "config file content")
	})

	t.Run("profile with value override", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_advanced.toml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// SSL_CERT should use direct value in dev
		gt.Equal(t, varMap["SSL_CERT"], "-----BEGIN CERTIFICATE-----\ndev-cert-content\n-----END CERTIFICATE-----")
	})

	t.Run("profile with command execution", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_advanced.toml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// HOSTNAME should execute command in staging
		gt.Equal(t, varMap["HOSTNAME"], "staging-server")
	})

	t.Run("profile with alias", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_advanced.toml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// API_KEY should resolve to SECONDARY_DB in prod
		gt.Equal(t, varMap["API_KEY"], "postgres://localhost/secondary")
	})

	t.Run("profile with template", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_advanced.toml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// CONFIG_PATH should use template in prod
		gt.Equal(t, varMap["CONFIG_PATH"], "/opt/default/myapp/config.yml")
	})
}

func TestProfileCompatibility(t *testing.T) {
	t.Run("backward compatibility with no profile", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_compat.toml", "", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// Non-profile variables should work as before
		gt.Equal(t, varMap["SIMPLE_VAR"], "simple_value")
		gt.Equal(t, varMap["COMPLEX_VAR"], "complex_value")
		gt.Equal(t, varMap["FILE_VAR"], "config file content")

		// Profile variables use default
		gt.Equal(t, varMap["PROFILE_VAR"], "default_profile_value")
		gt.Equal(t, varMap["MIXED_VAR"], "default_mixed")
	})

	t.Run("profile selection with mixed variables", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_compat.toml", "dev", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// Non-profile variables unchanged
		gt.Equal(t, varMap["SIMPLE_VAR"], "simple_value")
		gt.Equal(t, varMap["COMPLEX_VAR"], "complex_value")
		gt.Equal(t, varMap["FILE_VAR"], "config file content")

		// Profile variable uses dev value
		gt.Equal(t, varMap["PROFILE_VAR"], "dev_profile_value")
		// MIXED_VAR uses default (no dev profile)
		gt.Equal(t, varMap["MIXED_VAR"], "default_mixed")
	})

	t.Run("profile with partial coverage", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_compat.toml", "staging", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// MIXED_VAR has staging profile
		gt.Equal(t, varMap["MIXED_VAR"], "staging_mixed")
		// PROFILE_VAR uses default (no staging profile)
		gt.Equal(t, varMap["PROFILE_VAR"], "default_profile_value")
	})
}

func TestProfileNestedValidation(t *testing.T) {
	t.Run("nested profile in TOML should be rejected", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_nested.toml", "", nil)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("nested profile is not allowed")
	})
}
