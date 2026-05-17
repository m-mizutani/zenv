package executor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/zenv/v2/pkg/model"
	"golang.org/x/term"
)

func NewDefaultExecutor(opts Options) ExecuteFunc {
	return func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error {
		logger := ctxlog.From(ctx)
		logger.Debug("executing command", "cmd", cmd, "args", args, "env_vars", len(envVars))

		command := exec.CommandContext(ctx, cmd, args...)

		env := os.Environ()
		for _, envVar := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
		}
		command.Env = env

		if !opts.Redact {
			return runDirect(ctx, command)
		}

		var secrets []string
		for _, envVar := range envVars {
			if envVar.Secret && envVar.Value != "" {
				secrets = append(secrets, envVar.Value)
			}
		}

		// When the parent stdout is a real tty and the platform supports it,
		// route through a pty so the child sees a tty on its stdout. Otherwise
		// fall back to the pipe path (also used on Windows).
		if ptySupported() && len(secrets) > 0 && stdoutIsTTY() {
			return runWithPty(ctx, command, secrets)
		}

		return runWithPipe(ctx, command, secrets)
	}
}

// runDirect runs the command with the parent's stdin/stdout/stderr fds attached
// directly. Use this when redaction is disabled — the child inherits the tty.
func runDirect(ctx context.Context, command *exec.Cmd) error {
	logger := ctxlog.From(ctx)

	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return wrapExitError(logger, command, err)
	}
	logger.Debug("command executed successfully", "cmd", command.Path)
	return nil
}

// runWithPipe runs the command with anonymous pipes between the child's
// stdout/stderr and the parent's, going through redactWriter to mask any
// secret values. The child's stdout fd is a pipe, so isatty(fd) is false.
func runWithPipe(ctx context.Context, command *exec.Cmd, secrets []string) error {
	logger := ctxlog.From(ctx)

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

	if stdoutRedactor != nil {
		if flushErr := stdoutRedactor.Flush(); flushErr != nil {
			logger.Warn("failed to flush stdout redactor", "error", flushErr)
		}
	}
	if stderrRedactor != nil {
		if flushErr := stderrRedactor.Flush(); flushErr != nil {
			logger.Warn("failed to flush stderr redactor", "error", flushErr)
		}
	}

	if err != nil {
		return wrapExitError(logger, command, err)
	}
	logger.Debug("command executed successfully", "cmd", command.Path)
	return nil
}

// stdoutIsTTY reports whether os.Stdout is a tty. Used to decide whether the
// pty path is worth taking; if the parent's stdout is already redirected to a
// file, the pty path would inject ANSI escapes into that file.
func stdoutIsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// wrapExitError converts a *exec.ExitError into a model.ExecutorError. If the
// failure happened BEFORE the child started (e.g. binary not found or not
// executable), it is classified into a model.CommandLaunchError instead so the
// cli layer can render a helpful message.
func wrapExitError(logger *slog.Logger, command *exec.Cmd, err error) error {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		// (*exec.ExitError).ExitCode reports the child's exit status in a
		// cross-platform way (no need to type-assert to syscall.WaitStatus)
		// and returns -1 when the child was killed by a signal.
		exitCode := exitError.ExitCode()
		logger.Debug("command exited with non-zero code", "cmd", command.Path, "exit_code", exitCode)
		return model.NewExecutorError(err, exitCode)
	}

	logger.Debug("failed to launch command", "cmd", command.Path, "error", err)
	return classifyLaunchError(command.Args, err)
}

// classifyLaunchError converts an os/exec failure that occurred BEFORE the
// child started into a typed CommandLaunchError. The classification looks at
// the error chain rather than parsing the message string.
func classifyLaunchError(cmdAndArgs []string, err error) error {
	reason := model.LaunchOther
	switch {
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		reason = model.LaunchNotFound
	case errors.Is(err, fs.ErrPermission), errors.Is(err, syscall.ENOEXEC):
		reason = model.LaunchPermissionDenied
	}

	return &model.CommandLaunchError{
		Command: cmdAndArgs,
		Reason:  reason,
		Cause:   err,
	}
}
