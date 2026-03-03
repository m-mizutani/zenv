package executor_test

import (
	"bytes"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
)

// writeAndFlush is a test helper that writes data and flushes the redact writer.
func writeAndFlush(t *testing.T, w *executor.RedactWriterForTest, data string) int {
	t.Helper()
	n, err := w.Write([]byte(data))
	gt.NoError(t, err)
	gt.NoError(t, w.Flush())
	return n
}

func TestRedactWriter(t *testing.T) {
	t.Run("redact single secret value", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"my-secret-token"})

		n := writeAndFlush(t, w, "token is my-secret-token here")
		gt.Equal(t, n, len("token is my-secret-token here"))
		gt.Equal(t, buf.String(), "token is ***** here")
	})

	t.Run("redact multiple secret values", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"secret1", "secret2"})

		n := writeAndFlush(t, w, "a=secret1 b=secret2")
		gt.Equal(t, n, len("a=secret1 b=secret2"))
		gt.Equal(t, buf.String(), "a=***** b=*****")
	})

	t.Run("pass through when no secret matches", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"secret"})

		n := writeAndFlush(t, w, "nothing to redact")
		gt.Equal(t, n, len("nothing to redact"))
		gt.Equal(t, buf.String(), "nothing to redact")
	})

	t.Run("pass through with empty secret list", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{})

		n := writeAndFlush(t, w, "no secrets configured")
		gt.Equal(t, n, len("no secrets configured"))
		gt.Equal(t, buf.String(), "no secrets configured")
	})

	t.Run("longer secret replaced when shorter is substring", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"secret", "secret-long-value"})

		n := writeAndFlush(t, w, "val=secret-long-value")
		gt.Equal(t, n, len("val=secret-long-value"))
		gt.Equal(t, buf.String(), "val=*****")
	})

	t.Run("redact secret appearing multiple times", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"tok"})

		n := writeAndFlush(t, w, "tok and tok again")
		gt.Equal(t, n, len("tok and tok again"))
		gt.Equal(t, buf.String(), "***** and ***** again")
	})

	t.Run("secret split across two Write calls", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"secret123"})

		// Split "secret123" across two writes: "secre" + "t123"
		n1, err := w.Write([]byte("value=secre"))
		gt.NoError(t, err)
		gt.Equal(t, n1, len("value=secre"))

		n2, err := w.Write([]byte("t123 done"))
		gt.NoError(t, err)
		gt.Equal(t, n2, len("t123 done"))

		gt.NoError(t, w.Flush())

		gt.Equal(t, buf.String(), "value=***** done")
	})

	t.Run("secret at the very end with flush", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"token"})

		n, err := w.Write([]byte("my-token"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("my-token"))

		gt.NoError(t, w.Flush())

		gt.Equal(t, buf.String(), "my-*****")
	})

	t.Run("multiple writes without secret boundary", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"abc"})

		_, err := w.Write([]byte("hello "))
		gt.NoError(t, err)
		_, err = w.Write([]byte("world "))
		gt.NoError(t, err)
		_, err = w.Write([]byte("abc end"))
		gt.NoError(t, err)

		gt.NoError(t, w.Flush())

		gt.Equal(t, buf.String(), "hello world ***** end")
	})

	t.Run("byte-by-byte writes still redact secret", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"SECRET"})

		// Write "has SECRET inside" one byte at a time
		input := "has SECRET inside"
		for i := range input {
			n, err := w.Write([]byte{input[i]})
			gt.NoError(t, err)
			gt.Equal(t, n, 1)
		}

		gt.NoError(t, w.Flush())

		gt.S(t, buf.String()).NotContains("SECRET")
		gt.Equal(t, buf.String(), "has ***** inside")
	})

	t.Run("byte-by-byte writes with multiple secrets", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"AA", "BBB"})

		input := "xAAyBBBz"
		for i := range input {
			_, err := w.Write([]byte{input[i]})
			gt.NoError(t, err)
		}

		gt.NoError(t, w.Flush())

		gt.Equal(t, buf.String(), "x*****y*****z")
	})

	t.Run("byte-by-byte writes with no matching secret", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"NOPE"})

		input := "hello world"
		for i := range input {
			_, err := w.Write([]byte{input[i]})
			gt.NoError(t, err)
		}

		gt.NoError(t, w.Flush())

		gt.Equal(t, buf.String(), "hello world")
	})

	t.Run("small write entirely buffered then flushed", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"longersecret"})

		// Write data shorter than maxSecretLen - should be fully buffered
		n, err := w.Write([]byte("ok"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("ok"))
		gt.Equal(t, buf.String(), "") // nothing flushed yet

		gt.NoError(t, w.Flush())
		gt.Equal(t, buf.String(), "ok")
	})
}
