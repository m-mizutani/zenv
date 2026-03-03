package executor

import (
	"bytes"
	"io"
	"sort"
)

const redactMask = "*****"

var redactMaskBytes = []byte(redactMask)

type redactWriter struct {
	dest         io.Writer
	secrets      [][]byte
	maxSecretLen int
	buf          []byte
}

func newRedactWriter(dest io.Writer, secrets []string) *redactWriter {
	// Sort by length descending for longest-match-first semantics.
	sorted := make([]string, len(secrets))
	copy(sorted, secrets)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	secretBytes := make([][]byte, 0, len(sorted))
	maxLen := 0
	for _, s := range sorted {
		if len(s) == 0 {
			continue
		}
		secretBytes = append(secretBytes, []byte(s))
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}

	return &redactWriter{
		dest:         dest,
		secrets:      secretBytes,
		maxSecretLen: maxLen,
	}
}

// findEarliestMatch finds the earliest (leftmost) occurrence of any secret in
// data. When multiple secrets match at the same position, the longest wins.
// Returns (-1, 0) if no match is found.
func (w *redactWriter) findEarliestMatch(data []byte) (pos int, length int) {
	bestPos := -1
	bestLen := 0
	for _, secret := range w.secrets {
		idx := bytes.Index(data, secret)
		if idx == -1 {
			continue
		}
		if bestPos == -1 || idx < bestPos || (idx == bestPos && len(secret) > bestLen) {
			bestPos = idx
			bestLen = len(secret)
		}
	}
	return bestPos, bestLen
}

func (w *redactWriter) Write(p []byte) (int, error) {
	if w.maxSecretLen == 0 {
		_, err := w.dest.Write(p)
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}

	w.buf = append(w.buf, p...)

	var output []byte
	for len(w.buf) >= w.maxSecretLen {
		pos, length := w.findEarliestMatch(w.buf)

		if pos == -1 {
			// No match anywhere. Flush everything except the tail that
			// could be the beginning of a secret split across writes.
			safeEnd := len(w.buf) - (w.maxSecretLen - 1)
			output = append(output, w.buf[:safeEnd]...)
			w.buf = w.buf[safeEnd:]
			break
		}

		// Emit everything before the match, then the mask.
		output = append(output, w.buf[:pos]...)
		output = append(output, redactMaskBytes...)
		w.buf = w.buf[pos+length:]
	}

	if len(output) > 0 {
		if _, err := w.dest.Write(output); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

// Flush writes any remaining buffered data to the destination.
// Must be called after the child process finishes to ensure all output is emitted.
func (w *redactWriter) Flush() error {
	if len(w.buf) == 0 {
		return nil
	}

	// Process remaining buffer, replacing any complete matches.
	var output []byte
	for {
		pos, length := w.findEarliestMatch(w.buf)
		if pos == -1 {
			output = append(output, w.buf...)
			break
		}
		output = append(output, w.buf[:pos]...)
		output = append(output, redactMaskBytes...)
		w.buf = w.buf[pos+length:]
	}

	w.buf = nil
	if len(output) > 0 {
		_, err := w.dest.Write(output)
		return err
	}
	return nil
}
