package infra

import (
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

type Client interface {
	Exec(vars []*model.EnvVar, args types.Arguments) error
	ReadFile(filename types.FilePath) ([]byte, error)
	Prompt(msg string) string
	Stdout(format string, v ...interface{})
	PutKeyChainValues(envVars []*model.EnvVar, ns types.Namespace) error
	GetKeyChainValues(ns types.Namespace) ([]*model.EnvVar, error)
	ListKeyChainNamespaces(prefix types.NamespacePrefix) ([]types.Namespace, error)
	DeleteKeyChainValue(types.Namespace, types.EnvKey) error
}

type client struct{}

func New() *client {
	return &client{}
}
