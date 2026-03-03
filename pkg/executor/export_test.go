package executor

import "io"

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
