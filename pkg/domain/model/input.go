package model

type ExecInput struct {
	EnvVars []*EnvVar
	Args    []string
}

type ListInput struct {
	EnvVars []*EnvVar
	Args    []string
}

type WriteInput struct {
	Namespace string
	Args      []string
}
