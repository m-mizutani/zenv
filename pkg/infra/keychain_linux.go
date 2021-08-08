// +build linux

package infra

import (
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func (x *Infrastructure) PutKeyChainValues(envVars []*model.EnvVar, namespace string) error {
	return goerr.Wrap(model.ErrKeychainNotSupported)
}

func (x *Infrastructure) GetKeyChainValues(namespace string) ([]*model.EnvVar, error) {
	return nil, goerr.Wrap(model.ErrKeychainNotSupported)
}
