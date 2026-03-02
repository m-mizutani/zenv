package executor

import "io"

// NewRedactWriterForTest exposes newRedactWriter for testing.
func NewRedactWriterForTest(dest io.Writer, secrets []string) io.Writer {
	return newRedactWriter(dest, secrets)
}
