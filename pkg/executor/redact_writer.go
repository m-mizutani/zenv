package executor

import (
	"io"
	"sort"
	"strings"
)

const redactMask = "*****"

type redactWriter struct {
	dest    io.Writer
	secrets []string
}

func newRedactWriter(dest io.Writer, secrets []string) *redactWriter {
	// Sort secrets by length descending so longer values are replaced first.
	// This prevents partial matches when a shorter secret is a substring of a longer one.
	sorted := make([]string, len(secrets))
	copy(sorted, secrets)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	return &redactWriter{
		dest:    dest,
		secrets: sorted,
	}
}

func (w *redactWriter) Write(p []byte) (int, error) {
	s := string(p)
	for _, secret := range w.secrets {
		s = strings.ReplaceAll(s, secret, redactMask)
	}

	_, err := w.dest.Write([]byte(s))
	if err != nil {
		return 0, err
	}

	// Return original length so the caller (child process) does not think
	// it wrote fewer bytes than intended.
	return len(p), nil
}
