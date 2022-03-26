package model

import "github.com/m-mizutani/zenv/pkg/domain/types"

type ExecInput struct {
	EnvVars []*EnvVar
	Args    types.Arguments
}

type ListInput struct {
	EnvVars []*EnvVar
	Args    types.Arguments
}

type WriteSecretInput struct {
	Namespace types.NamespaceSuffix
	Key       types.EnvKey
}

type GenerateSecretInput struct {
	Namespace types.NamespaceSuffix
	Key       types.EnvKey
	Length    int64
}
