//go:build linux
// +build linux

package infra

import (
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func (x *client) PutKeyChainValues(envVars []*model.EnvVar, namespace string) error {
	return goerr.Wrap(types.ErrKeychainNotSupported)
}

func (x *client) GetKeyChainValues(namespace string) ([]*model.EnvVar, error) {
	return nil, goerr.Wrap(types.ErrKeychainNotSupported)
}
