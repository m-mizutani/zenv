package loader

// Test-only seams. These expose internal helpers and a hook to substitute the
// secret provider so tests never reach real cloud APIs.

var (
	SplitSecretFragment = splitSecretFragment
	ExtractJSONField    = extractJSONField
	ARNRegion           = arnRegion
)

// SetSecretProvider overrides the default secret provider used by resolvers and
// returns a function that restores the previous one. Intended for tests only.
func SetSecretProvider(p SecretProvider) (restore func()) {
	prev := newSecretProvider
	newSecretProvider = func() SecretProvider { return p }
	return func() { newSecretProvider = prev }
}
