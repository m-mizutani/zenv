package loader

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/m-mizutani/goerr/v2"
)

// SecretProvider fetches raw secret values from managed secret stores. The
// returned string is the secret payload as stored; JSON field extraction and
// reference template expansion are handled by the caller, not the provider.
type SecretProvider interface {
	GetAWSSecret(ctx context.Context, secretID string) (string, error)
	GetGCPSecret(ctx context.Context, name string) (string, error)
}

// newSecretProvider builds the default, SDK-backed secret provider. It is a
// variable so tests can substitute a mock without reaching real cloud APIs.
var newSecretProvider = func() SecretProvider { return &sdkSecretProvider{} }

// splitSecretFragment separates a secret reference of the form
// "<path>[#<json_key>]" into its path and optional JSON key. The first '#'
// delimits the fragment; secret names, ARNs and GCP resource paths never
// contain '#', so a plain first-occurrence split is unambiguous.
func splitSecretFragment(ref string) (path string, jsonKey string) {
	if path, key, found := strings.Cut(ref, "#"); found {
		return path, key
	}
	return ref, ""
}

// extractJSONField parses raw as a JSON object and returns the value at key as
// a string. String fields are returned verbatim; other scalars (numbers,
// booleans) and nested structures are rendered to their JSON representation so
// that, e.g., {"port":5432} yields "5432". It errors when raw is not a JSON
// object or the key is absent.
func extractJSONField(raw, key string) (string, error) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "", goerr.Wrap(err, "secret value is not a JSON object", goerr.V("key", key))
	}
	val, ok := obj[key]
	if !ok {
		return "", goerr.New("JSON key not found in secret value", goerr.V("key", key))
	}
	if val == nil {
		return "", nil
	}
	if s, ok := val.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(val)
	if err != nil {
		return "", goerr.Wrap(err, "failed to marshal JSON field value", goerr.V("key", key))
	}
	return string(b), nil
}

// isAWSSecretARN reports whether ref is a Secrets Manager ARN. Bare secret
// names are rejected at resolution time so the target region and account are
// always explicit.
func isAWSSecretARN(ref string) bool {
	return strings.HasPrefix(ref, "arn:aws:secretsmanager:")
}

// arnRegion extracts the region segment from an AWS ARN. It returns an empty
// string when ref is not an ARN.
func arnRegion(ref string) string {
	if !strings.HasPrefix(ref, "arn:") {
		return ""
	}
	// arn:partition:service:region:account-id:resource...
	parts := strings.SplitN(ref, ":", 6)
	if len(parts) < 4 {
		return ""
	}
	return parts[3]
}

// sdkSecretProvider is the production SecretProvider. Each cloud client is
// lazily initialized on first use so that configurations without any secret
// reference perform no network or credential resolution at all.
type sdkSecretProvider struct {
	awsOnce   sync.Once
	awsClient *secretsmanager.Client
	awsErr    error

	gcpOnce   sync.Once
	gcpClient *secretmanager.Client
	gcpErr    error
}

func (p *sdkSecretProvider) initAWS(ctx context.Context) error {
	p.awsOnce.Do(func() {
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			p.awsErr = goerr.Wrap(err, "failed to load AWS configuration")
			return
		}
		p.awsClient = secretsmanager.NewFromConfig(cfg)
	})
	return p.awsErr
}

func (p *sdkSecretProvider) GetAWSSecret(ctx context.Context, secretID string) (string, error) {
	if err := p.initAWS(ctx); err != nil {
		return "", err
	}

	// When the reference is an ARN that names a region, honor it per-operation
	// so a secret in a non-default region resolves without extra configuration.
	var optFns []func(*secretsmanager.Options)
	if region := arnRegion(secretID); region != "" {
		optFns = append(optFns, func(o *secretsmanager.Options) { o.Region = region })
	}

	out, err := p.awsClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}, optFns...)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get secret value from AWS Secrets Manager")
	}

	if out.SecretString != nil {
		return *out.SecretString, nil
	}
	if out.SecretBinary != nil {
		return string(out.SecretBinary), nil
	}
	return "", goerr.New("AWS secret has neither a string nor binary value")
}

func (p *sdkSecretProvider) initGCP(ctx context.Context) error {
	p.gcpOnce.Do(func() {
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			p.gcpErr = goerr.Wrap(err, "failed to create GCP Secret Manager client")
			return
		}
		p.gcpClient = client
	})
	return p.gcpErr
}

func (p *sdkSecretProvider) GetGCPSecret(ctx context.Context, name string) (string, error) {
	if err := p.initGCP(ctx); err != nil {
		return "", err
	}

	// Accept a secret path without an explicit version by defaulting to the
	// latest version, mirroring the gcloud CLI convenience.
	if !strings.Contains(name, "/versions/") {
		name = strings.TrimSuffix(name, "/") + "/versions/latest"
	}

	result, err := p.gcpClient.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", goerr.Wrap(err, "failed to access secret version in GCP Secret Manager")
	}
	if result.GetPayload() == nil {
		return "", goerr.New("GCP secret version has no payload")
	}
	return string(result.GetPayload().GetData()), nil
}
