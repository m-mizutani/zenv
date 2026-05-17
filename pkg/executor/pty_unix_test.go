//go:build !windows

package executor_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// captureStdout runs fn while os.Stdout is redirected to an os.Pipe and
// returns whatever was written. It restores os.Stdout afterwards.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	gt.NoError(t, err)

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()

	fn()

	gt.NoError(t, w.Close())
	return string(<-done)
}

func TestRunWithPty_ChildSeesTTY(t *testing.T) {
	if !executor.PtySupportedForTest() {
		t.Skip("pty not supported on this platform")
	}

	cmd := exec.Command("sh", "-c", "if [ -t 1 ]; then echo TTY_YES; else echo TTY_NO; fi")

	output := captureStdout(t, func() {
		err := executor.RunWithPtyForTest(context.Background(), cmd, nil)
		gt.NoError(t, err)
	})

	gt.S(t, output).Contains("TTY_YES")
	gt.S(t, output).NotContains("TTY_NO")
}

func TestRunWithPty_RedactsSecretInOutput(t *testing.T) {
	if !executor.PtySupportedForTest() {
		t.Skip("pty not supported on this platform")
	}

	cmd := exec.Command("sh", "-c", "echo my-secret-789 visible")

	output := captureStdout(t, func() {
		err := executor.RunWithPtyForTest(context.Background(), cmd, []string{"my-secret-789"})
		gt.NoError(t, err)
	})

	gt.S(t, output).NotContains("my-secret-789")
	gt.S(t, output).Contains("*****")
	gt.S(t, output).Contains("visible")
}

func TestRunWithPty_PreservesANSIEscape(t *testing.T) {
	if !executor.PtySupportedForTest() {
		t.Skip("pty not supported on this platform")
	}

	// Emit a red "hello" surrounded by ANSI escape sequences. The redactor
	// must pass the escape bytes through unchanged.
	cmd := exec.Command("sh", "-c", `printf '\033[31mhello\033[0m\n'`)

	output := captureStdout(t, func() {
		err := executor.RunWithPtyForTest(context.Background(), cmd, nil)
		gt.NoError(t, err)
	})

	gt.S(t, output).Contains("\x1b[31m")
	gt.S(t, output).Contains("hello")
	gt.S(t, output).Contains("\x1b[0m")
}

func TestRunWithPty_RedactsSecretEmbeddedInANSI(t *testing.T) {
	if !executor.PtySupportedForTest() {
		t.Skip("pty not supported on this platform")
	}

	// Wrap the secret in ANSI escape so we exercise the case where the
	// match is bracketed by control bytes.
	cmd := exec.Command("sh", "-c", `printf '\033[31mtopsecret\033[0m\n'`)

	output := captureStdout(t, func() {
		err := executor.RunWithPtyForTest(context.Background(), cmd, []string{"topsecret"})
		gt.NoError(t, err)
	})

	gt.S(t, output).NotContains("topsecret")
	gt.S(t, output).Contains("*****")
}

func TestRunWithPty_PropagatesExitCode(t *testing.T) {
	if !executor.PtySupportedForTest() {
		t.Skip("pty not supported on this platform")
	}

	cmd := exec.Command("sh", "-c", "exit 42")

	_ = captureStdout(t, func() {
		err := executor.RunWithPtyForTest(context.Background(), cmd, nil)
		gt.Error(t, err)
		gt.Equal(t, model.GetExitCode(err), 42)
		gt.True(t, model.IsExecutorError(err))
	})
}
