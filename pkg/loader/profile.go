package loader

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// CollectProfileNames reads the configuration file at path (YAML or HCL based
// on extension) and returns the set of profile names that are defined under
// any of its variables. The result is used by the CLI layer to validate that
// a user-supplied --profile flag actually maps to something.
//
// Behavior:
//   - If the file does not exist, an empty set is returned with a nil error.
//     The caller treats this as "this file contributed no profile names" and
//     decides separately whether the overall preflight should fail.
//   - If the file exists but cannot be parsed or is schema-invalid, the
//     underlying loader error is returned as-is (typically
//     *model.ConfigFileError) so the user sees the same diagnostics as during
//     the normal load.
//   - Unknown extensions are routed to the YAML reader, matching the loader
//     dispatcher in pkg/cli.
func CollectProfileNames(ctx context.Context, path string) (map[string]struct{}, error) {
	names := make(map[string]struct{})

	cfg, err := loadConfigForProfileScan(ctx, path)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return names, nil
	}

	for _, value := range cfg {
		for profileName := range value.Profile {
			names[profileName] = struct{}{}
		}
	}
	return names, nil
}

// loadConfigForProfileScan loads a YAMLConfig from disk without resolving any
// variables. The configuration is only used to enumerate profile names.
// mustExist is false: callers want a profile preflight to silently skip
// missing files, not to double-report file-absence errors.
func loadConfigForProfileScan(ctx context.Context, path string) (model.YAMLConfig, error) {
	if strings.EqualFold(filepath.Ext(path), ".hcl") {
		return loadHCLFile(ctx, path, false)
	}
	return loadAndMergeYAMLFiles(ctx, path, false)
}
