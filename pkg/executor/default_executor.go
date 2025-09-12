package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
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

		// Set up standard streams
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		err := command.Run()
		if err != nil {
			// Extract exit code
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					exitCode := status.ExitStatus()
					logger.Warn("command exited with non-zero code", "cmd", cmd, "exit_code", exitCode)
					return model.WithExitCode(goerr.Wrap(err, "command exited with non-zero code"), exitCode)
				}
			}
			logger.Error("failed to execute command", "cmd", cmd, "error", err)
			return model.WithExitCode(goerr.Wrap(err, "failed to execute command"), 1)
		}

		logger.Debug("command executed successfully", "cmd", cmd)
		return nil
	}
}
