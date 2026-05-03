package loader

import (
	"os"
	"path/filepath"
)

var (
	defaultDotEnvFiles = []string{".env"}
	defaultYAMLFiles   = []string{".env.yaml", ".env.yml"}
	defaultHCLFiles    = []string{".env.hcl"}
)

// FindFileUpward searches for a file by traversing parent directories.
// It starts from startDir and walks upward until it finds the file or reaches the root.
// Multiple filenames can be provided; the first match in each directory is returned.
// Returns the found file path, or empty string if not found.
func FindFileUpward(startDir string, filenames ...string) string {
	dir := filepath.Clean(startDir)

	for {
		for _, filename := range filenames {
			candidate := filepath.Join(dir, filename)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// ResolveDefaultDotEnvPath returns the default .env file path,
// searching parent directories from the current working directory.
func ResolveDefaultDotEnvPath() string {
	return resolveDefault(defaultDotEnvFiles)
}

// ResolveDefaultYAMLPath returns the default YAML config file path,
// searching parent directories from the current working directory.
func ResolveDefaultYAMLPath() string {
	return resolveDefault(defaultYAMLFiles)
}

// ResolveDefaultHCLPath returns the default HCL config file path,
// searching parent directories from the current working directory.
// Returns the fallback filename if not found.
func ResolveDefaultHCLPath() string {
	return resolveDefault(defaultHCLFiles)
}

// FindDefaultHCLPath returns the discovered HCL config file path, or
// empty string if no .env.hcl exists in the working directory or its ancestors.
func FindDefaultHCLPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return FindFileUpward(wd, defaultHCLFiles...)
}

func resolveDefault(filenames []string) string {
	wd, err := os.Getwd()
	if err != nil {
		return filenames[0]
	}
	if found := FindFileUpward(wd, filenames...); found != "" {
		return found
	}
	return filenames[0]
}
