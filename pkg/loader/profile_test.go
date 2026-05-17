package loader_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
)

func TestCollectProfileNames(t *testing.T) {
	ctx := context.Background()

	t.Run("collects all profile names from YAML", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/profile_basic.yaml")).NoError(t)
		gt.Equal(t, len(names), 3)
		for _, want := range []string{"dev", "staging", "prod"} {
			if _, ok := names[want]; !ok {
				t.Fatalf("expected profile %q to be present", want)
			}
		}
	})

	t.Run("collects all profile names from HCL", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/profile.hcl")).NoError(t)
		gt.Equal(t, len(names), 3)
		for _, want := range []string{"dev", "staging", "prod"} {
			if _, ok := names[want]; !ok {
				t.Fatalf("expected profile %q to be present", want)
			}
		}
	})

	t.Run("returns empty set for YAML file without any profile blocks", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/valid.yaml")).NoError(t)
		gt.Equal(t, len(names), 0)
	})

	t.Run("returns empty set when YAML file is missing", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/does_not_exist.yaml")).NoError(t)
		gt.Equal(t, len(names), 0)
	})

	t.Run("returns empty set when HCL file is missing", func(t *testing.T) {
		names := gt.R1(loader.CollectProfileNames(ctx, "testdata/does_not_exist.hcl")).NoError(t)
		gt.Equal(t, len(names), 0)
	})

	t.Run("propagates parse errors for malformed YAML", func(t *testing.T) {
		_, err := loader.CollectProfileNames(ctx, "testdata/invalid_syntax.yaml")
		gt.Error(t, err)
	})

	t.Run("propagates parse errors for malformed HCL", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "broken.hcl")
		gt.NoError(t, os.WriteFile(path, []byte("FOO =\n"), 0o600))
		_, err := loader.CollectProfileNames(ctx, path)
		gt.Error(t, err)
	})
}
