package executor

import (
	"context"
	"io"
	"os/exec"
)

// RunWithPtyForTest exposes runWithPty for tests on platforms that support pty.
// On Windows, runWithPty is a stub; tests guarding on build tag should skip.
func RunWithPtyForTest(ctx context.Context, cmd *exec.Cmd, secrets []string) error {
	return runWithPty(ctx, cmd, secrets)
}

// PtySupportedForTest exposes ptySupported for tests.
func PtySupportedForTest() bool { return ptySupported() }

// RedactWriterForTest wraps redactWriter for testing.
type RedactWriterForTest struct {
	w *redactWriter
}

// NewRedactWriterForTest exposes newRedactWriter for testing.
func NewRedactWriterForTest(dest io.Writer, secrets []string) *RedactWriterForTest {
	return &RedactWriterForTest{w: newRedactWriter(dest, secrets)}
}

func (t *RedactWriterForTest) Write(p []byte) (int, error) {
	return t.w.Write(p)
}

// Flush exposes redactWriter.Flush for testing.
func (t *RedactWriterForTest) Flush() error {
	return t.w.Flush()
}
