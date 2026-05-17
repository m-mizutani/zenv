package cli_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/cli"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func TestFormatError_Nil(t *testing.T) {
	gt.Equal(t, cli.FormatError(nil, false, false), "")
}

func TestFormatError_PlainError(t *testing.T) {
	got := cli.FormatError(errors.New("boom"), false, false)
	gt.S(t, got).
		Contains("Error:").
		Contains("boom").
		NotContains("\x1b[")
}

// The realistic scenario from the spec: a HCL variable BACKSTREAM_HEADER
// that references GOOGLE_ID_TOKEN, whose value is produced by a failing
// gcloud command.
func TestFormatError_RealisticChain(t *testing.T) {
	cmdErr := &model.CommandExecError{
		Command:  []string{"gcloud", "auth", "print-identity-token"},
		ExitCode: 1,
		Stderr:   "ERROR: (gcloud.auth.print-identity-token) Reauth required",
		Cause:    errors.New("exit status 1"),
	}
	refErr := &model.ReferenceError{
		Ref:    "GOOGLE_ID_TOKEN",
		Reason: model.RefResolveFailed,
		Cause:  cmdErr,
	}
	varErr := &model.VariableError{
		Key:     "BACKSTREAM_HEADER",
		Path:    ".env.hcl",
		Profile: "dev",
		Cause:   refErr,
	}

	got := cli.FormatError(varErr, false, false)

	// Top line
	gt.S(t, got).Contains(`Failed to resolve variable "BACKSTREAM_HEADER"`)
	// Source line with path and profile
	gt.S(t, got).Contains(".env.hcl").Contains("profile: dev")
	// Reference and command surfaces
	gt.S(t, got).Contains(`Reference "GOOGLE_ID_TOKEN" could not be resolved`)
	gt.S(t, got).Contains("gcloud auth print-identity-token").Contains("exit 1")
	// stderr tail surfaces (truncated rendering, but content visible)
	gt.S(t, got).Contains("Reauth required")
	// Hint surfaces
	gt.S(t, got).Contains("Hint:").Contains("gcloud auth print-identity-token")
	// No ANSI when color=false
	gt.S(t, got).NotContains("\x1b[")
	// No goerr internals
	gt.S(t, got).NotContains("error.stacktrace").NotContains("Stacktrace:")
}

func TestFormatError_Source_ProfileOnlyOrPathOnly(t *testing.T) {
	t.Run("path only", func(t *testing.T) {
		err := &model.VariableError{Key: "X", Path: ".env.hcl", Cause: errors.New("boom")}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).Contains("Source: .env.hcl").NotContains("(")
	})
	t.Run("profile only", func(t *testing.T) {
		err := &model.VariableError{Key: "X", Profile: "dev", Cause: errors.New("boom")}
		got := cli.FormatError(err, false, false)
		// Should not produce stray closing paren like "profile: dev)".
		gt.S(t, got).Contains("Source: profile: dev").NotContains(")")
	})
	t.Run("both", func(t *testing.T) {
		err := &model.VariableError{Key: "X", Path: ".env.hcl", Profile: "dev", Cause: errors.New("boom")}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).Contains("Source: .env.hcl  (profile: dev)")
	})
	t.Run("neither", func(t *testing.T) {
		err := &model.VariableError{Key: "X", Cause: errors.New("boom")}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).NotContains("Source:")
	})
}

func TestFormatError_RefNotFound_WithSuggestion(t *testing.T) {
	err := &model.VariableError{
		Key: "FOO", Path: ".env.hcl",
		Cause: &model.ReferenceError{
			Ref:       "GOOGLE_ID_TOKN", // typo
			Reason:    model.RefNotFound,
			Available: []string{"GOOGLE_ID_TOKEN", "AWS_KEY", "OTHER"},
		},
	}
	got := cli.FormatError(err, false, false)
	gt.S(t, got).
		Contains("not defined").
		Contains("did you mean").
		Contains(`"GOOGLE_ID_TOKEN"`)
}

func TestFormatError_RefCircular_PrintsChain(t *testing.T) {
	err := &model.VariableError{
		Key: "A",
		Cause: &model.ReferenceError{
			Ref:    "C",
			Reason: model.RefCircular,
			Chain:  []string{"A", "B", "C", "A"},
		},
	}
	got := cli.FormatError(err, false, false)
	gt.S(t, got).
		Contains("Circular reference").
		Contains("A -> B -> C -> A").
		Contains("break the cycle")
}

func TestFormatError_ConfigFileParseError(t *testing.T) {
	err := &model.ConfigFileError{
		Path:   ".env.hcl",
		Format: model.FormatHCL,
		Reason: model.ReasonParseError,
		Detail: "line 12: missing brace",
	}
	got := cli.FormatError(err, false, false)
	gt.S(t, got).
		Contains("Cannot parse hcl").
		Contains(".env.hcl").
		Contains("line 12").
		Contains("Hint:")
}

func TestFormatError_VerboseAppendsGoerrTrace(t *testing.T) {
	innerCause := goerr.New("inner failure")
	err := &model.VariableError{
		Key: "X", Path: ".env.hcl",
		Cause: &model.ResolveError{
			Op: model.OpReadFile, Target: "/no/such/file", Cause: innerCause,
		},
	}
	verbose := cli.FormatError(err, true, false)
	terse := cli.FormatError(err, false, false)

	gt.S(t, verbose).Contains("--- debug ---")
	gt.S(t, terse).NotContains("--- debug ---")

	// Both should contain the friendly summary
	gt.S(t, terse).Contains(`Failed to resolve variable "X"`)
	gt.S(t, verbose).Contains(`Failed to resolve variable "X"`)
}

func TestFormatError_ColorEmitsAnsi(t *testing.T) {
	err := &model.VariableError{Key: "X", Cause: errors.New("boom")}
	got := cli.FormatError(err, false, true)
	gt.S(t, got).Contains("\x1b[")
}

// Snapshot the realistic output for the BACKSTREAM_HEADER scenario, so the
// formatting can be eyeballed via `go test -run TestFormatError_Snapshot -v`.
func TestFormatError_Snapshot(t *testing.T) {
	cmdErr := &model.CommandExecError{
		Command:  []string{"gcloud", "auth", "print-identity-token"},
		ExitCode: 1,
		Stderr:   "ERROR: (gcloud.auth.print-identity-token) Reauth required.\nPlease run:\n  $ gcloud auth login",
		Cause:    errors.New("exit status 1"),
	}
	refErr := &model.ReferenceError{
		Ref:    "GOOGLE_ID_TOKEN",
		Reason: model.RefResolveFailed,
		Cause:  cmdErr,
	}
	varErr := &model.VariableError{
		Key:     "BACKSTREAM_HEADER",
		Path:    ".env.hcl",
		Profile: "dev",
		Cause:   refErr,
	}

	t.Log("\n--- terse / no color ---\n" + cli.FormatError(varErr, false, false))
	t.Log("\n--- verbose / no color ---\n" + cli.FormatError(varErr, true, false))
}

func TestFormatError_CommandLaunchError(t *testing.T) {
	t.Run("not found surfaces summary, cause and hint", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"nonexistent", "--flag"},
			Reason:  model.LaunchNotFound,
			Cause:   errors.New(`exec: "nonexistent": executable file not found in $PATH`),
		}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Command "nonexistent" not found in PATH`).
			Contains(`executable file not found`).
			Contains("Hint:").
			Contains("PATH").
			NotContains("\x1b[")
	})

	t.Run("permission denied surfaces dedicated hint", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"/tmp/foo"},
			Reason:  model.LaunchPermissionDenied,
			Cause:   errors.New("permission denied"),
		}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Command "/tmp/foo" is not executable`).
			Contains("permission denied").
			Contains("chmod +x")
	})

	t.Run("other reason falls back to generic hint", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"foo"},
			Reason:  model.LaunchOther,
			Cause:   errors.New("some other oddity"),
		}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Command "foo" failed to launch`).
			Contains("some other oddity").
			Contains("Hint:")
	})

	t.Run("no extra Cause line when cause is nil", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"foo"},
			Reason:  model.LaunchNotFound,
		}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Command "foo" not found`).
			NotContains("Cause:")
	})

	t.Run("renders generic name for empty command vector", func(t *testing.T) {
		err := &model.CommandLaunchError{Reason: model.LaunchNotFound}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Command "command" not found`).
			NotContains(`Command ""`)
	})

	t.Run("renders generic name when first element is empty string", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{""},
			Reason:  model.LaunchPermissionDenied,
		}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Command "command" is not executable`).
			NotContains(`Command ""`)
	})
}

// Snapshot the launch-error rendering so the formatting can be inspected via
// `go test -run TestFormatError_LaunchSnapshot -v`.
func TestFormatError_LaunchSnapshot(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"nonexistent", "--flag"},
			Reason:  model.LaunchNotFound,
			Cause:   errors.New(`exec: "nonexistent": executable file not found in $PATH`),
		}
		t.Log("\n" + cli.FormatError(err, false, false))
	})

	t.Run("permission denied", func(t *testing.T) {
		err := &model.CommandLaunchError{
			Command: []string{"/tmp/foo"},
			Reason:  model.LaunchPermissionDenied,
			Cause:   errors.New("fork/exec /tmp/foo: permission denied"),
		}
		t.Log("\n" + cli.FormatError(err, false, false))
	})
}

func TestFormatError_ConfigFileNotFound(t *testing.T) {
	err := &model.ConfigFileError{
		Path:   "/etc/missing.yaml",
		Format: model.FormatYAML,
		Reason: model.ReasonNotFound,
	}
	got := cli.FormatError(err, false, false)
	gt.S(t, got).
		Contains("Missing yaml config file").
		Contains("/etc/missing.yaml").
		Contains("Hint:")
}

func TestFormatError_ProfileNotFound(t *testing.T) {
	t.Run("with available alternatives", func(t *testing.T) {
		err := &model.ProfileNotFoundError{
			Profile:   "prod",
			Available: []string{"dev", "staging"},
			Paths:     []string{".env.yaml"},
		}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Profile "prod" is not defined`).
			Contains(".env.yaml").
			Contains("dev, staging")
	})

	t.Run("with no configuration loaded", func(t *testing.T) {
		err := &model.ProfileNotFoundError{Profile: "prod"}
		got := cli.FormatError(err, false, false)
		gt.S(t, got).
			Contains(`Profile "prod" is not defined`).
			Contains("no configuration file was loaded")
	})
}

func TestFormatError_InvalidLogLevel(t *testing.T) {
	err := &model.InvalidLogLevelError{
		Value:   "foo",
		Allowed: []string{"debug", "info", "warn", "error"},
	}
	got := cli.FormatError(err, false, false)
	gt.S(t, got).
		Contains(`Invalid log level "foo"`).
		Contains("debug, info, warn, error")
}

func TestFormatError_TerseDoesNotLeakInternalWrapChain(t *testing.T) {
	// Reproduce the historical "failed to load: failed to resolve: failed to
	// build: failed to resolve reference: failed to execute command" leak.
	cmdErr := &model.CommandExecError{Command: []string{"x"}, ExitCode: 1}
	ref := &model.ReferenceError{Ref: "R", Reason: model.RefResolveFailed, Cause: cmdErr}
	res := &model.ResolveError{Op: model.OpBuildContext, Cause: ref}
	v := &model.VariableError{Key: "K", Path: "f", Cause: res}

	got := cli.FormatError(v, false, false)
	// The leak words from the historical message should NOT appear all at once.
	leakWords := []string{"failed to load", "failed to resolve variable", "failed to build template context"}
	hits := 0
	for _, w := range leakWords {
		if strings.Contains(got, w) {
			hits++
		}
	}
	gt.N(t, hits).Less(2)
}
