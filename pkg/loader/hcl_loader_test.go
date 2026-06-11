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

// newHCLLoader / newHCLLoaderWithProfile are mustExist=false wrappers used in
// existing test cases. Strict (mustExist=true) behavior is covered by
// dedicated test cases that call the production constructors directly.
func newHCLLoader(path string, existingVars ...[]*model.EnvVar) loader.LoadFunc {
	return loader.NewHCLLoader(path, false, existingVars...)
}

func newHCLLoaderWithProfile(path, profile string, existingVars ...[]*model.EnvVar) loader.LoadFunc {
	return loader.NewHCLLoaderWithProfile(path, profile, false, existingVars...)
}

func TestHCLLoaderBasic(t *testing.T) {
	t.Setenv("ZENV_TEST_HOME", "/home/zenv-test")

	loadFunc := newHCLLoader("testdata/basic.hcl")
	envVars := gt.R1(loadFunc(context.Background())).NoError(t)

	got := envVarMap(envVars)

	gt.Equal(t, got["DB_HOST"].Value, "localhost")
	gt.Equal(t, got["DB_USER"].Value, "admin")
	gt.Equal(t, got["GH_REPO"].Value, "ubie-inc/foo")

	gt.Equal(t, got["DB_PASS"].Value, "secret")
	gt.True(t, got["DB_PASS"].Secret)

	gt.Equal(t, got["SSL_CERT"].Value, "config file content")

	gt.Equal(t, got["GIT_SHA"].Value, "abc123")

	gt.Equal(t, got["APP_HOME"].Value, "/home/zenv-test")

	for _, ev := range envVars {
		gt.Equal(t, ev.Source, model.SourceHCL)
	}
}

func TestHCLLoaderProfile(t *testing.T) {
	t.Run("default profile (no flag)", func(t *testing.T) {
		loadFunc := newHCLLoader("testdata/profile.hcl")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		got := envVarMap(envVars)

		gt.Equal(t, got["API_URL"].Value, "https://api.example.com")
		gt.Equal(t, got["DEBUG_MODE"].Value, "false")
		gt.Equal(t, got["SSL_CERT"].Value, "config file content")
	})

	t.Run("dev profile (scalar attribute and structured block)", func(t *testing.T) {
		loadFunc := newHCLLoaderWithProfile("testdata/profile.hcl", "dev")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		got := envVarMap(envVars)

		gt.Equal(t, got["API_URL"].Value, "http://localhost:8080")
		gt.Equal(t, got["DEBUG_MODE"].Value, "true")
		gt.Equal(t, got["SSL_CERT"].Value, "dev-cert-content")
	})

	t.Run("prod profile unsets DEBUG_MODE via null", func(t *testing.T) {
		loadFunc := newHCLLoaderWithProfile("testdata/profile.hcl", "prod")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		got := envVarMap(envVars)

		_, exists := got["DEBUG_MODE"]
		gt.False(t, exists)

		// Other variables fall back to default
		gt.Equal(t, got["API_URL"].Value, "https://api.example.com")
	})
}

func TestHCLLoaderTemplate(t *testing.T) {
	loadFunc := newHCLLoader("testdata/template.hcl")
	envVars := gt.R1(loadFunc(context.Background())).NoError(t)
	got := envVarMap(envVars)

	gt.Equal(t, got["DATABASE_URL"].Value, "postgresql://admin@localhost/myapp")
	gt.Equal(t, got["API_ENDPOINT"].Value, "https://staging.api.example.com")
}

func TestHCLLoaderNonExistentFile(t *testing.T) {
	t.Run("mustExist=false silently skips", func(t *testing.T) {
		loadFunc := newHCLLoader("testdata/does_not_exist.hcl")
		envVars := gt.R1(loadFunc(context.Background())).NoError(t)
		gt.Nil(t, envVars)
	})

	t.Run("mustExist=true returns ReasonNotFound", func(t *testing.T) {
		loadFunc := loader.NewHCLLoader("testdata/does_not_exist.hcl", true)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
		var cfgErr *model.ConfigFileError
		gt.True(t, errors.As(err, &cfgErr))
		gt.Equal(t, cfgErr.Format, model.FormatHCL)
		gt.Equal(t, cfgErr.Reason, model.ReasonNotFound)
	})
}

func TestHCLLoaderSyntaxError(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "broken.hcl")
	gt.NoError(t, os.WriteFile(path, []byte("FOO = \n"), 0600))

	loadFunc := newHCLLoader(path)
	_, err := loadFunc(context.Background())
	gt.Error(t, err)

	var cfgErr *model.ConfigFileError
	gt.True(t, errors.As(err, &cfgErr))
	gt.Equal(t, cfgErr.Reason, model.ReasonParseError)
}

// A parse failure must point the user at the offending location with a source
// snippet and a caret, not just a one-line summary.
func TestHCLLoaderSyntaxErrorDetail(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "broken.hcl")
	// An unterminated string spanning two physical lines: HCL reports an
	// "Invalid multi-line string" with remediation prose.
	gt.NoError(t, os.WriteFile(path, []byte("FOO = \"bar\nBAZ = \"qux\"\n"), 0600))

	loadFunc := newHCLLoader(path)
	_, err := loadFunc(context.Background())
	gt.Error(t, err)

	var cfgErr *model.ConfigFileError
	gt.True(t, errors.As(err, &cfgErr))
	gt.Equal(t, cfgErr.Reason, model.ReasonParseError)

	// Location, the offending source line, the caret and HCL's remediation
	// prose must all be present.
	gt.S(t, cfgErr.Detail).
		Contains("line 1").
		Contains(`FOO = "bar`).
		Contains("^").
		Contains("multi-line")
}

func TestHCLLoaderDuplicateName(t *testing.T) {
	t.Run("attribute and block share a name", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "dup_attr_block.hcl")
		content := `FOO = "x"
FOO {
  value = "y"
}
`
		gt.NoError(t, os.WriteFile(path, []byte(content), 0600))

		loadFunc := newHCLLoader(path)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
	})

	t.Run("two blocks share a name", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "dup_block_block.hcl")
		content := `FOO {
  value = "x"
}
FOO {
  value = "y"
}
`
		gt.NoError(t, os.WriteFile(path, []byte(content), 0600))

		loadFunc := newHCLLoader(path)
		_, err := loadFunc(context.Background())
		gt.Error(t, err)
	})
}

func TestHCLLoaderConflictingValueTypes(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "conflict.hcl")
	content := `FOO {
  value = "x"
  file  = "/tmp/abc"
}
`
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))

	loadFunc := newHCLLoader(path)
	_, err := loadFunc(context.Background())
	gt.Error(t, err)
}

func TestHCLLoaderUnknownAttribute(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "unknown_attr.hcl")
	content := `FOO {
  value      = "x"
  unexpected = "boom"
}
`
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))

	loadFunc := newHCLLoader(path)
	_, err := loadFunc(context.Background())
	gt.Error(t, err)
}

func TestHCLLoaderBlockWithLabelRejected(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "labeled.hcl")
	content := `FOO "label" {
  value = "x"
}
`
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))

	loadFunc := newHCLLoader(path)
	_, err := loadFunc(context.Background())
	gt.Error(t, err)
}

func TestHCLLoaderInvalidRefs(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad_refs.hcl")
	content := `FOO {
  file = "/tmp/x"
  refs = ["BAR"]
}
`
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))

	loadFunc := newHCLLoader(path)
	_, err := loadFunc(context.Background())
	gt.Error(t, err)
}

func envVarMap(vars []*model.EnvVar) map[string]*model.EnvVar {
	m := make(map[string]*model.EnvVar, len(vars))
	for _, v := range vars {
		m[v.Name] = v
	}
	return m
}
