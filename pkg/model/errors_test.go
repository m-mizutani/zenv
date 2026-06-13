package model_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestConfigFileError(t *testing.T) {
	t.Run("reports format and path", func(t *testing.T) {
		err := &model.ConfigFileError{
			Path:   ".env.hcl",
			Format: model.FormatHCL,
			Reason: model.ReasonParseError,
			Detail: "line 12: missing brace",
		}
		gt.S(t, err.Error()).Contains("hcl").Contains(".env.hcl").Contains("line 12")
	})

	t.Run("unwraps cause", func(t *testing.T) {
		cause := errors.New("permission denied")
		err := &model.ConfigFileError{
			Path:   ".env.yaml",
			Format: model.FormatYAML,
			Reason: model.ReasonNotReadable,
			Cause:  cause,
		}
		gt.True(t, errors.Is(err, cause))
	})
}

func TestConfigFileError_NotFound(t *testing.T) {
	t.Run("reports missing file with path", func(t *testing.T) {
		err := &model.ConfigFileError{
			Path:   "missing.yaml",
			Format: model.FormatYAML,
			Reason: model.ReasonNotFound,
		}
		gt.S(t, err.Error()).
			Contains("missing").
			Contains("yaml").
			Contains("missing.yaml")
	})

	t.Run("reason String", func(t *testing.T) {
		gt.Equal(t, model.ReasonNotFound.String(), "not found")
	})
}

func TestProfileNotFoundError(t *testing.T) {
	t.Run("with available profiles", func(t *testing.T) {
		err := &model.ProfileNotFoundError{
			Profile:   "prod",
			Available: []string{"dev", "staging"},
			Paths:     []string{".env.yaml"},
		}
		gt.S(t, err.Error()).
			Contains(`"prod"`).
			Contains("not found").
			Contains("dev, staging")
	})

	t.Run("with no profiles defined anywhere", func(t *testing.T) {
		err := &model.ProfileNotFoundError{
			Profile: "prod",
			Paths:   []string{".env.yaml"},
		}
		gt.S(t, err.Error()).
			Contains(`"prod"`).
			Contains("no profiles defined")
	})

	t.Run("with no configuration file loaded", func(t *testing.T) {
		err := &model.ProfileNotFoundError{Profile: "prod"}
		gt.S(t, err.Error()).
			Contains(`"prod"`).
			Contains("no configuration file")
	})
}

func TestInvalidLogLevelError(t *testing.T) {
	t.Run("includes allowed list", func(t *testing.T) {
		err := &model.InvalidLogLevelError{
			Value:   "foo",
			Allowed: []string{"debug", "info", "warn", "error"},
		}
		gt.S(t, err.Error()).
			Contains(`"foo"`).
			Contains("debug").
			Contains("info").
			Contains("warn").
			Contains("error")
	})

	t.Run("without allowed list", func(t *testing.T) {
		err := &model.InvalidLogLevelError{Value: "foo"}
		gt.S(t, err.Error()).
			Contains(`"foo"`).
			Contains("invalid log level")
	})
}

func TestVariableError(t *testing.T) {
	t.Run("carries key in message", func(t *testing.T) {
		err := &model.VariableError{
			Key:   "API_KEY",
			Path:  ".env.hcl",
			Cause: errors.New("boom"),
		}
		gt.S(t, err.Error()).Contains("API_KEY").Contains("boom")
	})

	t.Run("can be extracted via errors.As", func(t *testing.T) {
		inner := &model.VariableError{Key: "FOO"}
		wrapped := &model.ConfigFileError{Cause: inner}
		var got *model.VariableError
		gt.True(t, errors.As(wrapped, &got))
		gt.Equal(t, got.Key, "FOO")
	})
}

func TestResolveError(t *testing.T) {
	t.Run("reports op kind", func(t *testing.T) {
		err := &model.ResolveError{
			Op:     model.OpExecCommand,
			Target: "gcloud auth print-identity-token",
			Cause:  errors.New("exit status 1"),
		}
		gt.S(t, err.Error()).
			Contains("execute command").
			Contains("gcloud").
			Contains("exit status 1")
	})

	t.Run("unwraps cause", func(t *testing.T) {
		cause := errors.New("io error")
		err := &model.ResolveError{Op: model.OpReadFile, Cause: cause}
		gt.True(t, errors.Is(err, cause))
	})
}

func TestReferenceError(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		err := &model.ReferenceError{Ref: "FOO", Reason: model.RefNotFound}
		gt.S(t, err.Error()).Contains("FOO").Contains("not found")
	})

	t.Run("circular reports chain", func(t *testing.T) {
		err := &model.ReferenceError{
			Ref:    "A",
			Reason: model.RefCircular,
			Chain:  []string{"A", "B", "A"},
		}
		gt.S(t, err.Error()).Contains("A -> B -> A").Contains("circular")
	})

	t.Run("resolve failed unwraps cause", func(t *testing.T) {
		cause := errors.New("inner")
		err := &model.ReferenceError{Ref: "X", Reason: model.RefResolveFailed, Cause: cause}
		gt.True(t, errors.Is(err, cause))
	})
}

func TestCommandExecError(t *testing.T) {
	t.Run("formats command and exit code", func(t *testing.T) {
		err := &model.CommandExecError{
			Command:  []string{"gcloud", "auth", "print-identity-token"},
			ExitCode: 1,
		}
		gt.S(t, err.Error()).
			Contains("gcloud auth print-identity-token").
			Contains("status 1")
	})

	t.Run("unwraps cause", func(t *testing.T) {
		cause := errors.New("exec failed")
		err := &model.CommandExecError{Command: []string{"x"}, Cause: cause}
		gt.True(t, errors.Is(err, cause))
	})
}

func TestSecretError(t *testing.T) {
	t.Run("formats provider and ref", func(t *testing.T) {
		err := &model.SecretError{
			Provider: model.SecretProviderAWS,
			Ref:      "prod/db/password",
		}
		gt.S(t, err.Error()).
			Contains("AWS Secrets Manager").
			Contains("prod/db/password")
	})

	t.Run("includes json key when present", func(t *testing.T) {
		err := &model.SecretError{
			Provider: model.SecretProviderGCP,
			Ref:      "projects/p/secrets/s/versions/latest",
			JSONKey:  "host",
		}
		gt.S(t, err.Error()).
			Contains("GCP Secret Manager").
			Contains("host")
	})

	t.Run("unwraps cause", func(t *testing.T) {
		cause := errors.New("access denied")
		err := &model.SecretError{Provider: model.SecretProviderAWS, Ref: "x", Cause: cause}
		gt.True(t, errors.Is(err, cause))
	})
}

func TestCommandLaunchError(t *testing.T) {
	t.Run("not-found message includes executable name", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"nonexistent", "arg1"},
			Reason:  model.LaunchNotFound,
		}
		gt.S(t, err.Error()).
			Contains("nonexistent").
			Contains("not found").
			Contains("PATH")
	})

	t.Run("permission-denied message includes executable name", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"/tmp/not-executable"},
			Reason:  model.LaunchPermissionDenied,
		}
		gt.S(t, err.Error()).
			Contains("/tmp/not-executable").
			Contains("not executable")
	})

	t.Run("other reason includes underlying cause", func(t *testing.T) {
		cause := errors.New("operation not permitted")
		err := &model.CommandLaunchError{
			Command: []string{"foo"},
			Reason:  model.LaunchOther,
			Cause:   cause,
		}
		gt.S(t, err.Error()).
			Contains("foo").
			Contains("operation not permitted")
	})

	t.Run("unwraps cause", func(t *testing.T) {
		cause := errors.New("inner")
		err := &model.CommandLaunchError{Command: []string{"x"}, Cause: cause}
		gt.True(t, errors.Is(err, cause))
	})

	t.Run("exit code follows POSIX shell convention", func(t *testing.T) {
		gt.Equal(t,
			(&model.CommandLaunchError{Reason: model.LaunchNotFound}).ExitCode(),
			127,
		)
		gt.Equal(t,
			(&model.CommandLaunchError{Reason: model.LaunchPermissionDenied}).ExitCode(),
			126,
		)
		gt.Equal(t,
			(&model.CommandLaunchError{Reason: model.LaunchOther}).ExitCode(),
			1,
		)
	})

	t.Run("can be extracted via errors.As through a wrapper", func(t *testing.T) {
		inner := &model.CommandLaunchError{Command: []string{"foo"}, Reason: model.LaunchNotFound}
		wrapped := &model.ResolveError{Op: model.OpExecCommand, Cause: inner}
		var got *model.CommandLaunchError
		gt.True(t, errors.As(wrapped, &got))
		gt.Equal(t, got.Reason, model.LaunchNotFound)
	})

	t.Run("falls back to generic name when command vector is empty", func(t *testing.T) {
		err := &model.CommandLaunchError{Reason: model.LaunchNotFound}
		gt.S(t, err.Error()).
			Contains(`"command"`).
			NotContains(`""`)
	})

	t.Run("falls back to generic name when first element is empty string", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"", "arg"},
			Reason:  model.LaunchPermissionDenied,
		}
		gt.S(t, err.Error()).
			Contains(`"command"`).
			NotContains(`""`)
	})
}

func TestTruncateStderr(t *testing.T) {
	t.Run("keeps short input intact", func(t *testing.T) {
		gt.Equal(t, model.TruncateStderr("hello", 100), "hello")
	})

	t.Run("keeps tail when exceeding limit", func(t *testing.T) {
		long := strings.Repeat("a", 50) + "END"
		got := model.TruncateStderr(long, 5)
		gt.Equal(t, len(got), 5)
		gt.S(t, got).HasSuffix("aaEND")
	})

	t.Run("non-positive limit returns input unchanged", func(t *testing.T) {
		gt.Equal(t, model.TruncateStderr("abc", 0), "abc")
	})
}

func TestNestedChainTraversal(t *testing.T) {
	cmd := &model.CommandExecError{
		Command:  []string{"gcloud", "auth", "print-identity-token"},
		ExitCode: 1,
		Cause:    errors.New("exit status 1"),
	}
	inner := &model.ResolveError{Op: model.OpExecCommand, Target: "gcloud", Cause: cmd}
	ref := &model.ReferenceError{Ref: "GOOGLE_ID_TOKEN", Reason: model.RefResolveFailed, Cause: inner}
	outer := &model.VariableError{Key: "BACKSTREAM_HEADER", Path: ".env.hcl", Profile: "dev", Cause: ref}

	var gotCmd *model.CommandExecError
	gt.True(t, errors.As(outer, &gotCmd))
	gt.Equal(t, gotCmd.ExitCode, 1)

	var gotRef *model.ReferenceError
	gt.True(t, errors.As(outer, &gotRef))
	gt.Equal(t, gotRef.Ref, "GOOGLE_ID_TOKEN")
}
