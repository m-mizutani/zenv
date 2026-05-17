package cli

// IsTruthyEnvForTest exposes isTruthyEnv for testing.
func IsTruthyEnvForTest(v string) bool { return isTruthyEnv(v) }
