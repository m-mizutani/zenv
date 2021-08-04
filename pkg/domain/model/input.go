package model

type ExecInput struct {
	EnvVars []*EnvVar
	Args    []string
}

type WriteInput struct {
	Namespace string
	EnvVar
}
