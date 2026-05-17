//go:build !windows

package executor

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
)

// ptySupported reports whether the current platform can run the pty path.
func ptySupported() bool { return true }

// runWithPty starts command attached to a newly allocated pty pair and writes
// the pty output through a redactWriter to the parent's stdout. The child's
// stdin/stdout/stderr all go through the pty slave so the child sees a real
// tty (isatty == true, winsize ioctls work, etc.).
//
// The output copy runs in the foreground while cmd.Wait runs in a background
// goroutine. This ordering guarantees that io.Copy enters its first read on
// the pty master before any teardown can close it, which avoids a startup
// race that previously dropped the child's output on fast-completing
// commands.
func runWithPty(ctx context.Context, command *exec.Cmd, secrets []string) error {
	logger := ctxlog.From(ctx)

	ptmx, err := pty.Start(command)
	if err != nil {
		return goerr.Wrap(err, "failed to start pty")
	}
	defer func() {
		if cerr := ptmx.Close(); cerr != nil && !errors.Is(cerr, os.ErrClosed) {
			logger.Warn("failed to close pty master", "error", cerr)
		}
	}()

	// Propagate window-size changes from the parent's stdin to the pty.
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	winchStop := make(chan struct{})
	winchDone := make(chan struct{})
	go func() {
		defer close(winchDone)
		for {
			select {
			case <-winch:
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					logger.Warn("failed to inherit pty size", "error", err)
				}
			case <-winchStop:
				return
			}
		}
	}()
	// Trigger initial sizing.
	winch <- syscall.SIGWINCH

	redactor := newRedactWriter(os.Stdout, secrets)

	// Best-effort stdin pump. May remain blocked on os.Stdin.Read after the
	// child exits because os.Stdin can't be cancelled cleanly; this is the
	// standard pty pattern (zenv exits shortly after, releasing it).
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Run cmd.Wait in the background. We drive the output copy in the
	// foreground so it is guaranteed to be inside io.Copy's first read by
	// the time the child exits.
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- command.Wait()
	}()

	_, copyErr := io.Copy(redactor, ptmx)
	if copyErr != nil &&
		!errors.Is(copyErr, io.EOF) &&
		!errors.Is(copyErr, os.ErrClosed) &&
		!isPtyClosedReadError(copyErr) {
		logger.Warn("pty output copy reported error", "error", copyErr)
	}

	waitErr := <-waitDone

	signal.Stop(winch)
	close(winchStop)
	<-winchDone

	if flushErr := redactor.Flush(); flushErr != nil {
		logger.Warn("failed to flush pty redactor", "error", flushErr)
	}

	if waitErr != nil {
		return wrapExitError(logger, command, waitErr)
	}
	logger.Debug("command executed successfully", "cmd", command.Path)
	return nil
}

// isPtyClosedReadError reports whether err is the EIO read returned by Linux
// when the slave side of a pty has been closed by the child exiting.
func isPtyClosedReadError(err error) bool {
	var pErr *os.PathError
	if errors.As(err, &pErr) {
		return errors.Is(pErr.Err, syscall.EIO)
	}
	return errors.Is(err, syscall.EIO)
}
