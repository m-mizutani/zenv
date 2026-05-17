package model

import (
	"fmt"
	"strings"
)

// ConfigFormat identifies the syntax of a configuration file.
type ConfigFormat int

const (
	FormatYAML ConfigFormat = iota
	FormatHCL
	FormatDotEnv
)

func (f ConfigFormat) String() string {
	switch f {
	case FormatYAML:
		return "yaml"
	case FormatHCL:
		return "hcl"
	case FormatDotEnv:
		return "dotenv"
	default:
		return "unknown"
	}
}

// ConfigFileReason classifies why a configuration file failed.
type ConfigFileReason int

const (
	// ReasonNotReadable indicates the file could not be opened or read.
	ReasonNotReadable ConfigFileReason = iota
	// ReasonParseError indicates a syntactic parse failure.
	ReasonParseError
	// ReasonInvalidSchema indicates a semantically invalid configuration
	// (unknown attribute, mutually exclusive fields, duplicate names, ...).
	ReasonInvalidSchema
)

func (r ConfigFileReason) String() string {
	switch r {
	case ReasonNotReadable:
		return "not readable"
	case ReasonParseError:
		return "parse error"
	case ReasonInvalidSchema:
		return "invalid schema"
	default:
		return "unknown"
	}
}

// ConfigFileError describes a failure related to loading or interpreting a
// configuration file.
type ConfigFileError struct {
	Path   string
	Format ConfigFormat
	Reason ConfigFileReason
	// Detail carries parser-provided context (line numbers, diagnostics, ...).
	Detail string
	Cause  error
}

func (e *ConfigFileError) Error() string {
	var b strings.Builder
	switch e.Reason {
	case ReasonNotReadable:
		b.WriteString("cannot read ")
	case ReasonParseError:
		b.WriteString("cannot parse ")
	case ReasonInvalidSchema:
		b.WriteString("invalid ")
	default:
		b.WriteString("error in ")
	}
	b.WriteString(e.Format.String())
	if e.Path != "" {
		b.WriteString(" file ")
		b.WriteString(e.Path)
	} else {
		b.WriteString(" file")
	}
	if e.Detail != "" {
		b.WriteString(": ")
		b.WriteString(e.Detail)
	}
	return b.String()
}

func (e *ConfigFileError) Unwrap() error { return e.Cause }

// VariableError attaches the surrounding processing context (which variable in
// which file under which profile) to an underlying failure. It carries no
// failure semantics of its own; the actual reason lives in Cause.
type VariableError struct {
	Key     string
	Path    string
	Profile string
	Cause   error
}

func (e *VariableError) Error() string {
	var b strings.Builder
	b.WriteString("variable ")
	if e.Key != "" {
		fmt.Fprintf(&b, "%q ", e.Key)
	}
	b.WriteString("failed")
	if e.Cause != nil {
		fmt.Fprintf(&b, ": %v", e.Cause)
	}
	return b.String()
}

func (e *VariableError) Unwrap() error { return e.Cause }

// ResolveOp classifies the kind of value-resolution operation that failed.
type ResolveOp int

const (
	OpReadFile ResolveOp = iota
	OpExecCommand
	OpTemplate
	OpAlias
	OpBuildContext
)

func (o ResolveOp) String() string {
	switch o {
	case OpReadFile:
		return "read file"
	case OpExecCommand:
		return "execute command"
	case OpTemplate:
		return "expand template"
	case OpAlias:
		return "resolve alias"
	case OpBuildContext:
		return "build template context"
	default:
		return "unknown"
	}
}

// ResolveError describes a failure of a specific value-resolution step.
type ResolveError struct {
	Op ResolveOp
	// Target is a short description of what the operation acted on
	// (file path, command rendering, alias name, etc.).
	Target string
	Cause  error
}

func (e *ResolveError) Error() string {
	var b strings.Builder
	b.WriteString(e.Op.String())
	if e.Target != "" {
		fmt.Fprintf(&b, " %q", e.Target)
	}
	b.WriteString(" failed")
	if e.Cause != nil {
		fmt.Fprintf(&b, ": %v", e.Cause)
	}
	return b.String()
}

func (e *ResolveError) Unwrap() error { return e.Cause }

// ReferenceReason classifies why a reference lookup failed.
type ReferenceReason int

const (
	// RefNotFound indicates the referenced name has no definition.
	RefNotFound ReferenceReason = iota
	// RefCircular indicates a cycle was detected while resolving a reference.
	RefCircular
	// RefResolveFailed indicates the referenced variable exists but its own
	// resolution failed.
	RefResolveFailed
)

func (r ReferenceReason) String() string {
	switch r {
	case RefNotFound:
		return "not found"
	case RefCircular:
		return "circular reference"
	case RefResolveFailed:
		return "resolution failed"
	default:
		return "unknown"
	}
}

// ReferenceError describes a failed reference lookup.
type ReferenceError struct {
	Ref    string
	Reason ReferenceReason
	// Chain holds the offending cycle for RefCircular (ordered).
	Chain []string
	// Available holds candidate names for sugar-suggesting on RefNotFound.
	Available []string
	Cause     error
}

func (e *ReferenceError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "reference %q ", e.Ref)
	switch e.Reason {
	case RefNotFound:
		b.WriteString("not found")
	case RefCircular:
		b.WriteString("forms a circular reference")
		if len(e.Chain) > 0 {
			fmt.Fprintf(&b, " (%s)", strings.Join(e.Chain, " -> "))
		}
	case RefResolveFailed:
		b.WriteString("could not be resolved")
		if e.Cause != nil {
			fmt.Fprintf(&b, ": %v", e.Cause)
		}
	default:
		b.WriteString("failed")
	}
	return b.String()
}

func (e *ReferenceError) Unwrap() error { return e.Cause }

// CommandExecError describes a failure of an external command spawned to
// produce a configuration value (not the final command zenv was asked to run).
type CommandExecError struct {
	Command  []string
	ExitCode int
	// Stderr holds the tail of the failed command's stderr output (may be empty).
	Stderr string
	Cause  error
}

func (e *CommandExecError) Error() string {
	cmd := strings.Join(e.Command, " ")
	if e.ExitCode != 0 {
		return fmt.Sprintf("command %q exited with status %d", cmd, e.ExitCode)
	}
	if e.Cause != nil {
		return fmt.Sprintf("command %q failed: %v", cmd, e.Cause)
	}
	return fmt.Sprintf("command %q failed", cmd)
}

func (e *CommandExecError) Unwrap() error { return e.Cause }

// LaunchReason classifies why the target command (the one zenv was asked to
// run) could not be launched.
type LaunchReason int

const (
	// LaunchNotFound indicates the executable was not found in PATH.
	LaunchNotFound LaunchReason = iota
	// LaunchPermissionDenied indicates the executable was found but cannot be
	// executed (e.g. missing executable bit, not a regular file).
	LaunchPermissionDenied
	// LaunchOther covers any other startup failure that prevented the child
	// process from ever running.
	LaunchOther
)

func (r LaunchReason) String() string {
	switch r {
	case LaunchNotFound:
		return "not found"
	case LaunchPermissionDenied:
		return "permission denied"
	case LaunchOther:
		return "launch failed"
	default:
		return "unknown"
	}
}

// CommandLaunchError describes a failure to start the target command that
// zenv was asked to run. Because the child process never executed, its stderr
// did not appear and zenv itself is responsible for reporting the failure.
type CommandLaunchError struct {
	// Command is the full command vector including arguments. The executable
	// name is Command[0].
	Command []string
	Reason  LaunchReason
	Cause   error
}

func (e *CommandLaunchError) Error() string {
	name := ""
	if len(e.Command) > 0 {
		name = e.Command[0]
	}
	switch e.Reason {
	case LaunchNotFound:
		return fmt.Sprintf("command %q not found in PATH", name)
	case LaunchPermissionDenied:
		return fmt.Sprintf("command %q is not executable", name)
	default:
		if e.Cause != nil {
			return fmt.Sprintf("command %q failed to launch: %v", name, e.Cause)
		}
		return fmt.Sprintf("command %q failed to launch", name)
	}
}

func (e *CommandLaunchError) Unwrap() error { return e.Cause }

// ExitCode returns the exit code that zenv should propagate when the target
// command failed to launch. Values follow the POSIX shell convention used by
// bash: 127 for not-found, 126 for permission/exec failures, 1 otherwise.
func (e *CommandLaunchError) ExitCode() int {
	switch e.Reason {
	case LaunchNotFound:
		return 127
	case LaunchPermissionDenied:
		return 126
	default:
		return 1
	}
}

// TruncateStderr returns the trailing portion of s, never exceeding limit
// bytes. The result is what should be stored in CommandExecError.Stderr.
func TruncateStderr(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[len(s)-limit:]
}
