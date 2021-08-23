package infra

import "github.com/m-mizutani/zenv/pkg/domain/model"

type Interface interface {
	Exec(vars []*model.EnvVar, args []string) error
	ReadFile(filename string) ([]byte, error)
	Prompt(msg string) string
	Stdout(format string, v ...interface{})
	PutKeyChainValues(envVars []*model.EnvVar, namespace string) error
	GetKeyChainValues(namespace string) ([]*model.EnvVar, error)
}
