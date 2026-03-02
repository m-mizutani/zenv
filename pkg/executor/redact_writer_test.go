package executor_test

import (
	"bytes"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/v2/pkg/executor"
)

func TestRedactWriter(t *testing.T) {
	t.Run("redact single secret value", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"my-secret-token"})

		n, err := w.Write([]byte("token is my-secret-token here"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("token is my-secret-token here"))
		gt.Equal(t, buf.String(), "token is ***** here")
	})

	t.Run("redact multiple secret values", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"secret1", "secret2"})

		n, err := w.Write([]byte("a=secret1 b=secret2"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("a=secret1 b=secret2"))
		gt.Equal(t, buf.String(), "a=***** b=*****")
	})

	t.Run("pass through when no secret matches", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"secret"})

		n, err := w.Write([]byte("nothing to redact"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("nothing to redact"))
		gt.Equal(t, buf.String(), "nothing to redact")
	})

	t.Run("pass through with empty secret list", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{})

		n, err := w.Write([]byte("no secrets configured"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("no secrets configured"))
		gt.Equal(t, buf.String(), "no secrets configured")
	})

	t.Run("longer secret replaced first to avoid partial match", func(t *testing.T) {
		var buf bytes.Buffer
		// "secret-long-value" contains "secret" as a substring.
		// The longer value must be replaced first so that it is fully masked.
		w := executor.NewRedactWriterForTest(&buf, []string{"secret", "secret-long-value"})

		n, err := w.Write([]byte("val=secret-long-value"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("val=secret-long-value"))
		gt.Equal(t, buf.String(), "val=*****")
	})

	t.Run("redact secret appearing multiple times", func(t *testing.T) {
		var buf bytes.Buffer
		w := executor.NewRedactWriterForTest(&buf, []string{"tok"})

		n, err := w.Write([]byte("tok and tok again"))
		gt.NoError(t, err)
		gt.Equal(t, n, len("tok and tok again"))
		gt.Equal(t, buf.String(), "***** and ***** again")
	})
}
