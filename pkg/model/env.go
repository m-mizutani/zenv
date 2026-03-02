package model

import (
	"errors"
	"fmt"
)

type EnvVar struct {
	Name   string
	Value  string
	Source EnvSource
}

type EnvSource int

const (
	SourceSystem EnvSource = iota
	SourceDotEnv
	SourceYAML
	SourceInline
)

// ExecutorError represents an error from command execution.
// When the executed command exits with a non-zero code, its stderr output
// is already visible to the user, so zenv should not print additional messages.
type ExecutorError struct {
	err      error
	exitCode int
}

// NewExecutorError creates a new ExecutorError wrapping the given error
func NewExecutorError(err error, exitCode int) *ExecutorError {
	return &ExecutorError{err: err, exitCode: exitCode}
}

func (e *ExecutorError) Error() string {
	return fmt.Sprintf("command exited with code %d: %v", e.exitCode, e.err)
}

func (e *ExecutorError) Unwrap() error {
	return e.err
}

// ExitCode returns the exit code of the failed command
func (e *ExecutorError) ExitCode() int {
	return e.exitCode
}

// IsExecutorError checks whether the error originates from command execution
func IsExecutorError(err error) bool {
	var execErr *ExecutorError
	return errors.As(err, &execErr)
}

// GetExitCode extracts exit code from an error, returns 1 if not found
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}

	var execErr *ExecutorError
	if errors.As(err, &execErr) {
		return execErr.ExitCode()
	}

	return 1 // Default exit code for errors
}
