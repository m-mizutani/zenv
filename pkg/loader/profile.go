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
//   - When mustExist is true, a missing file is reported as
//     *model.ConfigFileError (Reason = ReasonNotFound). Callers pass true for
//     paths that came from an explicit --config flag so the profile preflight
//     surfaces the file-absence error rather than masking it as
//     "profile not found".
//   - When mustExist is false, a missing file returns (nil, nil). The caller
//     uses the nil result to learn that this path contributed nothing — both
//     to the profile-name union and to the "searched paths" list shown in
//     ProfileNotFoundError.
//   - If the file exists but cannot be parsed or is schema-invalid, the
//     underlying loader error is returned as-is (typically
//     *model.ConfigFileError) so the user sees the same diagnostics as during
//     the normal load.
//   - Unknown extensions are routed to the YAML reader, matching the loader
//     dispatcher in pkg/cli.
func CollectProfileNames(ctx context.Context, path string, mustExist bool) (map[string]struct{}, error) {
	cfg, err := loadConfigForProfileScan(ctx, path, mustExist)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, nil
	}

	names := make(map[string]struct{})
	for _, value := range cfg {
		for profileName := range value.Profile {
			names[profileName] = struct{}{}
		}
	}
	return names, nil
}

// loadConfigForProfileScan loads a YAMLConfig from disk without resolving any
// variables. The configuration is only used to enumerate profile names. The
// mustExist flag is forwarded to the underlying loader so that explicit
// --config paths fail loudly when the file is missing.
func loadConfigForProfileScan(ctx context.Context, path string, mustExist bool) (model.YAMLConfig, error) {
	if strings.EqualFold(filepath.Ext(path), ".hcl") {
		return loadHCLFile(ctx, path, mustExist)
	}
	return loadAndMergeYAMLFiles(ctx, path, mustExist)
}
