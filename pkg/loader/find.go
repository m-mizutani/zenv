package loader

import (
	"os"
	"path/filepath"
)

// FindFileUpward searches for a file by traversing parent directories.
// It starts from startDir and walks upward until it finds the file or reaches the root.
// Returns the found file path, or empty string if not found.
func FindFileUpward(startDir string, filename string) string {
	dir := filepath.Clean(startDir)

	for {
		candidate := filepath.Join(dir, filename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			return ""
		}
		dir = parent
	}
}
