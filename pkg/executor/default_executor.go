package executor

import (
	"context"
	"fmt"
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
			// Extract exit code
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode := status.ExitStatus()
					logger.Debug("command exited with non-zero code", "cmd", cmd, "exit_code", exitCode)
					return model.NewExecutorError(err, exitCode)
				}
			}
			logger.Debug("failed to execute command", "cmd", cmd, "error", err)
			return model.NewExecutorError(err, 1)
		}

		logger.Debug("command executed successfully", "cmd", cmd)
		return nil
	}
}
