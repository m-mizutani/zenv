package executor

import (
	"context"

	"github.com/m-mizutani/zenv/v2/pkg/model"
)

type ExecuteFunc func(ctx context.Context, cmd string, args []string, envVars []*model.EnvVar) error

// Options controls behavior of the default executor.
type Options struct {
	// Redact enables secret value redaction in the child process's
	// stdout/stderr. When false (default), the child inherits the parent's
	// fds directly so its tty characteristics (isatty, winsize, color, etc.)
	// are preserved.
	Redact bool
}
