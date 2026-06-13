package loader_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/loader"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// mockSecretProvider is an in-memory SecretProvider used to keep tests away
// from real cloud APIs. The maps are keyed by the secret path (the part before
// any "#json_key" fragment).
type mockSecretProvider struct {
	aws    map[string]string
	gcp    map[string]string
	awsErr error
	gcpErr error
}

func (m *mockSecretProvider) GetAWSSecret(_ context.Context, secretID string) (string, error) {
	if m.awsErr != nil {
		return "", m.awsErr
	}
	v, ok := m.aws[secretID]
	if !ok {
		return "", goerr.New("secret not found", goerr.V("id", secretID))
	}
	return v, nil
}

func (m *mockSecretProvider) GetGCPSecret(_ context.Context, name string) (string, error) {
	if m.gcpErr != nil {
		return "", m.gcpErr
	}
	v, ok := m.gcp[name]
	if !ok {
		return "", goerr.New("secret not found", goerr.V("name", name))
	}
	return v, nil
}

func TestSplitSecretFragment(t *testing.T) {
	t.Run("path with fragment", func(t *testing.T) {
		path, key := loader.SplitSecretFragment("prod/db/password#password")
		gt.Equal(t, path, "prod/db/password")
		gt.Equal(t, key, "password")
	})

	t.Run("path without fragment", func(t *testing.T) {
		path, key := loader.SplitSecretFragment("prod/db/password")
		gt.Equal(t, path, "prod/db/password")
		gt.Equal(t, key, "")
	})

	t.Run("resource path with fragment", func(t *testing.T) {
		path, key := loader.SplitSecretFragment("projects/p/secrets/s/versions/latest#token")
		gt.Equal(t, path, "projects/p/secrets/s/versions/latest")
		gt.Equal(t, key, "token")
	})
}

func TestExtractJSONField(t *testing.T) {
	t.Run("extracts string field", func(t *testing.T) {
		v := gt.R1(loader.ExtractJSONField(`{"user":"alice","password":"s3cret"}`, "password")).NoError(t)
		gt.Equal(t, v, "s3cret")
	})

	t.Run("non-JSON value errors", func(t *testing.T) {
		_, err := loader.ExtractJSONField("plain-text", "password")
		gt.Error(t, err)
	})

	t.Run("missing key errors", func(t *testing.T) {
		_, err := loader.ExtractJSONField(`{"user":"alice"}`, "password")
		gt.Error(t, err)
	})

	t.Run("non-string field errors", func(t *testing.T) {
		_, err := loader.ExtractJSONField(`{"port":5432}`, "port")
		gt.Error(t, err)
	})
}

func TestARNRegion(t *testing.T) {
	t.Run("extracts region from ARN", func(t *testing.T) {
		r := loader.ARNRegion("arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:prod/db/pw")
		gt.Equal(t, r, "ap-northeast-1")
	})

	t.Run("bare name has no region", func(t *testing.T) {
		gt.Equal(t, loader.ARNRegion("prod/db/password"), "")
	})
}

func TestIsAWSSecretARN(t *testing.T) {
	t.Run("full ARN is accepted", func(t *testing.T) {
		gt.True(t, loader.IsAWSSecretARN("arn:aws:secretsmanager:ap-northeast-1:123456789012:secret:prod/db/pw"))
	})

	t.Run("bare name is rejected", func(t *testing.T) {
		gt.False(t, loader.IsAWSSecretARN("prod/db/password"))
	})

	t.Run("non-secretsmanager ARN is rejected", func(t *testing.T) {
		gt.False(t, loader.IsAWSSecretARN("arn:aws:s3:::my-bucket/key"))
	})
}

// writeConfig writes content to a file under a fresh temp dir and returns its path.
func writeConfig(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestSecretResolutionYAML(t *testing.T) {
	const (
		pwARN   = "arn:aws:secretsmanager:ap-northeast-1:1:secret:prod/db/password"
		connARN = "arn:aws:secretsmanager:ap-northeast-1:1:secret:prod/db/conn"
	)
	provider := &mockSecretProvider{
		aws: map[string]string{
			pwARN:   "raw-aws-secret",
			connARN: `{"host":"db.example.com","port":"5432"}`,
		},
		gcp: map[string]string{
			"projects/p/secrets/token/versions/latest": "raw-gcp-secret",
		},
	}
	restore := loader.SetSecretProvider(provider)
	defer restore()

	t.Run("AWS raw secret via ARN", func(t *testing.T) {
		path := writeConfig(t, ".env.yaml", "DB_PW:\n  aws_secret: "+pwARN+"\n  secret: true\n")
		envVars := gt.R1(loader.NewYAMLLoader(path, true)(context.Background())).NoError(t)
		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "DB_PW")
		gt.Equal(t, envVars[0].Value, "raw-aws-secret")
		gt.Equal(t, envVars[0].Secret, true)
	})

	t.Run("AWS JSON field extraction", func(t *testing.T) {
		path := writeConfig(t, ".env.yaml", "DB_HOST:\n  aws_secret: "+connARN+"#host\n")
		envVars := gt.R1(loader.NewYAMLLoader(path, true)(context.Background())).NoError(t)
		gt.Equal(t, envVars[0].Value, "db.example.com")
	})

	t.Run("bare AWS name is rejected", func(t *testing.T) {
		path := writeConfig(t, ".env.yaml", "DB_PW:\n  aws_secret: prod/db/password\n")
		_, err := loader.NewYAMLLoader(path, true)(context.Background())
		gt.Error(t, err)
		var se *model.SecretError
		gt.True(t, errors.As(err, &se))
		gt.S(t, err.Error()).Contains("ARN")
	})

	t.Run("GCP resource path", func(t *testing.T) {
		path := writeConfig(t, ".env.yaml", "TOKEN:\n  gcp_secret: projects/p/secrets/token/versions/latest\n")
		envVars := gt.R1(loader.NewYAMLLoader(path, true)(context.Background())).NoError(t)
		gt.Equal(t, envVars[0].Value, "raw-gcp-secret")
	})

	t.Run("refs expand in secret path", func(t *testing.T) {
		path := writeConfig(t, ".env.yaml",
			"REGION:\n  value: ap-northeast-1\n"+
				"DB_PW:\n  aws_secret: \"arn:aws:secretsmanager:{{.REGION}}:1:secret:prod/db/password\"\n  refs: [REGION]\n")
		envVars := gt.R1(loader.NewYAMLLoader(path, true)(context.Background())).NoError(t)
		var dbpw string
		for _, e := range envVars {
			if e.Name == "DB_PW" {
				dbpw = e.Value
			}
		}
		gt.Equal(t, dbpw, "raw-aws-secret")
	})

	t.Run("fetch error surfaces as SecretError", func(t *testing.T) {
		missingARN := "arn:aws:secretsmanager:ap-northeast-1:1:secret:no/such/secret"
		path := writeConfig(t, ".env.yaml", "MISSING:\n  aws_secret: "+missingARN+"\n")
		_, err := loader.NewYAMLLoader(path, true)(context.Background())
		gt.Error(t, err)
		var se *model.SecretError
		gt.True(t, errors.As(err, &se))
		gt.Equal(t, se.Provider, model.SecretProviderAWS)
		gt.Equal(t, se.Ref, missingARN)
	})

	t.Run("JSON extraction error surfaces as SecretError", func(t *testing.T) {
		path := writeConfig(t, ".env.yaml", "X:\n  aws_secret: "+connARN+"#missing\n")
		_, err := loader.NewYAMLLoader(path, true)(context.Background())
		gt.Error(t, err)
		var se *model.SecretError
		gt.True(t, errors.As(err, &se))
		gt.Equal(t, se.JSONKey, "missing")
	})
}

func TestSecretResolutionHCL(t *testing.T) {
	const connARN = "arn:aws:secretsmanager:ap-northeast-1:1:secret:prod/db/conn"
	provider := &mockSecretProvider{
		aws: map[string]string{connARN: `{"host":"db.example.com"}`},
		gcp: map[string]string{"projects/p/secrets/token/versions/latest": "raw-gcp-secret"},
	}
	restore := loader.SetSecretProvider(provider)
	defer restore()

	t.Run("AWS secret with JSON key in HCL", func(t *testing.T) {
		path := writeConfig(t, ".env.hcl", "DB_HOST {\n  aws_secret = \""+connARN+"#host\"\n}\n")
		envVars := gt.R1(loader.NewHCLLoader(path, true)(context.Background())).NoError(t)
		gt.Equal(t, len(envVars), 1)
		gt.Equal(t, envVars[0].Name, "DB_HOST")
		gt.Equal(t, envVars[0].Value, "db.example.com")
		gt.Equal(t, envVars[0].Source, model.SourceHCL)
	})

	t.Run("GCP secret in HCL", func(t *testing.T) {
		path := writeConfig(t, ".env.hcl",
			"TOKEN {\n  gcp_secret = \"projects/p/secrets/token/versions/latest\"\n  secret = true\n}\n")
		envVars := gt.R1(loader.NewHCLLoader(path, true)(context.Background())).NoError(t)
		gt.Equal(t, envVars[0].Value, "raw-gcp-secret")
		gt.Equal(t, envVars[0].Secret, true)
	})
}

// TestGCPSecretIntegration exercises the real GCP Secret Manager SDK path. It
// runs only when both TEST_GCP_SECRET_KEY (the secret resource path) and
// TEST_GCP_SECRET_VALUE (its expected value) are present, and relies on the
// ambient Application Default Credentials. Without those env vars it is skipped.
func TestGCPSecretIntegration(t *testing.T) {
	key := os.Getenv("TEST_GCP_SECRET_KEY")
	want := os.Getenv("TEST_GCP_SECRET_VALUE")
	if key == "" || want == "" {
		t.Skip("TEST_GCP_SECRET_KEY and TEST_GCP_SECRET_VALUE are not set")
	}

	path := writeConfig(t, ".env.yaml", "GCP_SECRET:\n  gcp_secret: "+key+"\n  secret: true\n")
	envVars := gt.R1(loader.NewYAMLLoader(path, true)(context.Background())).NoError(t)
	gt.Equal(t, len(envVars), 1)
	gt.Equal(t, envVars[0].Name, "GCP_SECRET")
	gt.Equal(t, envVars[0].Value, want)
	gt.Equal(t, envVars[0].Secret, true)
}
