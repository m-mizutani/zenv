package executor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"syscall"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func NewDefaultExecutor() ExecuteFunc {
	return func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
		logger := ctxlog.From(ctx)
		logger.Debug("executing command", "cmd", cmd, "args", args, "env_vars", len(envVars))

		command := exec.CommandContext(ctx, cmd, args...)

		// Set environment variables
		env := os.Environ()
		for _, envVar := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
		}
		command.Env = env

		// Collect secret values for redaction
		var secrets []string
		for _, envVar := range envVars {
			if envVar.Secret && envVar.Value != "" {
				secrets = append(secrets, envVar.Value)
			}
		}

		// Set up standard streams with optional redaction
		command.Stdin = os.Stdin
		var stdoutRedactor, stderrRedactor *redactWriter
		if len(secrets) > 0 {
			stdoutRedactor = newRedactWriter(os.Stdout, secrets)
			stderrRedactor = newRedactWriter(os.Stderr, secrets)
			command.Stdout = stdoutRedactor
			command.Stderr = stderrRedactor
		} else {
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
		}

		err := command.Run()

		// Flush any remaining buffered data from redact writers
		if stdoutRedactor != nil {
			_ = stdoutRedactor.Flush()
		}
		if stderrRedactor != nil {
			_ = stderrRedactor.Flush()
		}
		if err != nil {
			// Case 1: the child actually ran and exited non-zero. Its stderr
			// is already on the user's terminal, so we wrap it as an
			// ExecutorError and stay silent at the cli layer.
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				exitCode := 1
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
				logger.Debug("command exited with non-zero code", "cmd", cmd, "exit_code", exitCode)
				return model.NewExecutorError(err, exitCode)
			}

			// Case 2: the child never ran. Classify why so the cli layer can
			// emit a helpful error message (the child has produced no stderr
			// of its own).
			logger.Debug("failed to launch command", "cmd", cmd, "error", err)
			return classifyLaunchError(cmd, args, err)
		}

		logger.Debug("command executed successfully", "cmd", cmd)
		return nil
	}
}

// classifyLaunchError converts an os/exec failure that occurred BEFORE the
// child started into a typed CommandLaunchError so the cli layer can render
// it. The classification looks at the error chain rather than parsing the
// message string.
func classifyLaunchError(cmd string, args []string, err error) error {
	command := append([]string{cmd}, args...)

	reason := model.LaunchOther
	switch {
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		reason = model.LaunchNotFound
	case errors.Is(err, fs.ErrPermission), errors.Is(err, syscall.ENOEXEC):
		reason = model.LaunchPermissionDenied
	}

	return &model.CommandLaunchError{
		Command: command,
		Reason:  reason,
		Cause:   err,
	}
}
