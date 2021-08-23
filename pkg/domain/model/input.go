package model

type ExecInput struct {
	EnvVars []*EnvVar
	Args    []string
}

type ListInput struct {
	EnvVars []*EnvVar
	Args    []string
}

type WriteSecretInput struct {
	Namespace string
	Key       string
}

type GenerateSecretInput struct {
	Namespace string
	Key       string
	Length    int64
}
