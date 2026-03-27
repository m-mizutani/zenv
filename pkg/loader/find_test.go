package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
)

func TestFindFileUpward(t *testing.T) {
	t.Run("file exists in start directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, ".env")
		f := gt.R1(os.Create(target)).NoError(t)
		gt.NoError(t, f.Close())

		result := loader.FindFileUpward(tmpDir, ".env")
		gt.Value(t, result).Equal(target)
	})

	t.Run("file exists in parent directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, ".env")
		f := gt.R1(os.Create(target)).NoError(t)
		gt.NoError(t, f.Close())

		child := filepath.Join(tmpDir, "subdir")
		gt.NoError(t, os.Mkdir(child, 0o755))

		result := loader.FindFileUpward(child, ".env")
		gt.Value(t, result).Equal(target)
	})

	t.Run("file exists two levels up", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, ".env.yaml")
		f := gt.R1(os.Create(target)).NoError(t)
		gt.NoError(t, f.Close())

		child := filepath.Join(tmpDir, "a", "b")
		gt.NoError(t, os.MkdirAll(child, 0o755))

		result := loader.FindFileUpward(child, ".env.yaml")
		gt.Value(t, result).Equal(target)
	})

	t.Run("multiple filenames finds first match", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, ".env.yml")
		f := gt.R1(os.Create(target)).NoError(t)
		gt.NoError(t, f.Close())

		child := filepath.Join(tmpDir, "subdir")
		gt.NoError(t, os.Mkdir(child, 0o755))

		result := loader.FindFileUpward(child, ".env.yaml", ".env.yml")
		gt.Value(t, result).Equal(target)
	})

	t.Run("multiple filenames prefers first in same directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlTarget := filepath.Join(tmpDir, ".env.yaml")
		f1 := gt.R1(os.Create(yamlTarget)).NoError(t)
		gt.NoError(t, f1.Close())
		ymlTarget := filepath.Join(tmpDir, ".env.yml")
		f2 := gt.R1(os.Create(ymlTarget)).NoError(t)
		gt.NoError(t, f2.Close())

		result := loader.FindFileUpward(tmpDir, ".env.yaml", ".env.yml")
		gt.Value(t, result).Equal(yamlTarget)
	})

	t.Run("file not found returns empty string", func(t *testing.T) {
		tmpDir := t.TempDir()
		child := filepath.Join(tmpDir, "deep", "nested")
		gt.NoError(t, os.MkdirAll(child, 0o755))

		result := loader.FindFileUpward(child, ".nonexistent")
		gt.Value(t, result).Equal("")
	})
}
