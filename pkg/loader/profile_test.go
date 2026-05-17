package loader_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestCollectProfileNames(t *testing.T) {
	ctx := context.Background()

	t.Run("collects all profile names from YAML", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/profile_basic.yaml", false)).NoError(t)
		gt.Equal(t, len(names), 3)
		for _, want := range []string{"dev", "staging", "prod"} {
			if _, ok := names[want]; !ok {
				t.Fatalf("expected profile %q to be present", want)
			}
		}
	})

	t.Run("collects all profile names from HCL", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/profile.hcl", false)).NoError(t)
		gt.Equal(t, len(names), 3)
		for _, want := range []string{"dev", "staging", "prod"} {
			if _, ok := names[want]; !ok {
				t.Fatalf("expected profile %q to be present", want)
			}
		}
	})

	t.Run("returns empty set for YAML file without any profile blocks", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/valid.yaml", false)).NoError(t)
		gt.Equal(t, len(names), 0)
	})

	t.Run("returns nil when YAML file is missing and mustExist=false", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/does_not_exist.yaml", false)).NoError(t)
		gt.Nil(t, names)
	})

	t.Run("returns nil when HCL file is missing and mustExist=false", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/does_not_exist.hcl", false)).NoError(t)
		gt.Nil(t, names)
	})

	t.Run("returns ConfigFileError when YAML file is missing and mustExist=true", func(t *testing.T) {
		_, err := loader.CollectProfileNames(ctx, "testdata/does_not_exist.yaml", true)
		gt.Error(t, err)
		var cfgErr *model.ConfigFileError
		gt.True(t, errors.As(err, &cfgErr))
		gt.Equal(t, cfgErr.Format, model.FormatYAML)
		gt.Equal(t, cfgErr.Reason, model.ReasonNotFound)
	})

	t.Run("returns ConfigFileError when HCL file is missing and mustExist=true", func(t *testing.T) {
		_, err := loader.CollectProfileNames(ctx, "testdata/does_not_exist.hcl", true)
		gt.Error(t, err)
		var cfgErr *model.ConfigFileError
		gt.True(t, errors.As(err, &cfgErr))
		gt.Equal(t, cfgErr.Format, model.FormatHCL)
		gt.Equal(t, cfgErr.Reason, model.ReasonNotFound)
	})

	t.Run("propagates parse errors for malformed YAML", func(t *testing.T) {
		_, err := loader.CollectProfileNames(ctx, "testdata/invalid_syntax.yaml", false)
		gt.Error(t, err)
	})

	t.Run("propagates parse errors for malformed HCL", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "broken.hcl")
		gt.NoError(t, os.WriteFile(path, []byte("FOO =\n"), 0o600))
		_, err := loader.CollectProfileNames(ctx, path, false)
		gt.Error(t, err)
	})
}
