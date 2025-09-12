package model

import "github.com/m-mizutani/goerr/v2"

type EnvVar struct {
	Name   string
	Value  string
	Source EnvSource
}

type EnvSource int

const (
	SourceSystem EnvSource = iota
	SourceDotEnv
	SourceTOML
	SourceInline
)

// ExitCode represents the exit code of a command execution
type ExitCode int

// ExitCodeKey is the key for storing exit codes in goerr.TypedValue
const ExitCodeKey = "exit_code"

// WithExitCode adds exit code to a goerr error
func WithExitCode(err error, code int) error {
	return goerr.Wrap(err, "command failed", goerr.V(ExitCodeKey, ExitCode(code)))
}

// GetExitCode extracts exit code from a goerr error, returns 1 if not found
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}

	if gErr, ok := err.(*goerr.Error); ok {
		if val := gErr.Values()[ExitCodeKey]; val != nil {
			if code, ok := val.(ExitCode); ok {
				return int(code)
			}
		}
	}

	return 1 // Default exit code for errors
}
