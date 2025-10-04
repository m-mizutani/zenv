package loader_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
)

func TestProfileNilHandling(t *testing.T) {
	// Create test data that could potentially return nil from GetValueForProfile
	t.Run("nil profile value should be skipped", func(t *testing.T) {
		// This tests the case where a profile returns nil
		// (though in practice GetValueForProfile always returns non-nil)
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_unset.toml", "nonexistent", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		// Should get default values since "nonexistent" profile doesn't exist
		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// These should all have their default values
		gt.Equal(t, varMap["DEBUG_FLAG"], "true")
		gt.Equal(t, varMap["VERBOSE_MODE"], "enabled")
		gt.Equal(t, varMap["FEATURE_FLAG"], "on")
	})

	t.Run("empty object unsets variable correctly", func(t *testing.T) {
		loadFunc := loader.NewTOMLLoaderWithProfile("testdata/profile_unset.toml", "prod", nil)
		envVars, err := loadFunc(context.Background())
		gt.NoError(t, err)

		varMap := make(map[string]string)
		for _, v := range envVars {
			varMap[v.Name] = v.Value
		}

		// DEBUG_FLAG and VERBOSE_MODE should not exist (unset by empty object)
		_, hasDebugFlag := varMap["DEBUG_FLAG"]
		_, hasVerboseMode := varMap["VERBOSE_MODE"]
		gt.False(t, hasDebugFlag)
		gt.False(t, hasVerboseMode)

		// FEATURE_FLAG should exist (no prod profile)
		gt.Equal(t, varMap["FEATURE_FLAG"], "on")
	})
}
