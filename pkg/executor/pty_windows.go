//go:build windows

package executor

import (
	"context"
	"os/exec"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// ptySupported reports whether the current platform can run the pty path.
// On Windows, creack/pty's API differs significantly from the Unix one, so the
// pty path is disabled and the pipe path is always used instead.
func ptySupported() bool { return false }

// runWithPty is unreachable on Windows because ptySupported() returns false;
// it exists only so the cross-platform dispatcher in default_executor.go
// compiles on this build target.
func runWithPty(_ context.Context, _ *exec.Cmd, _ []string) error {
	return model.NewExecutorError(goerr.New("pty path not supported on windows"), 1)
}
