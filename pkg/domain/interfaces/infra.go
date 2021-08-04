package interfaces

import "github.com/m-mizutani/zenv/pkg/domain/model"

type Infra interface {
	Exec(vars []*model.EnvVar, args []string) error
	Prompt(msg string) string
	PutKeyChainValues(envVars []*model.EnvVar, namespace string) error
	GetKeyChainValues(namespace string) ([]*model.EnvVar, error)
}
