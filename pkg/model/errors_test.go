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
