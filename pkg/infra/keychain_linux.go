//go:build linux
// +build linux

package infra

import (
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func (x *client) PutKeyChainValues(envVars []*model.EnvVar, ns types.Namespace) error {
	return goerr.Wrap(types.ErrKeychainNotSupported)
}
func (x *client) GetKeyChainValues(ns types.Namespace) ([]*model.EnvVar, error) {
	return nil, goerr.Wrap(types.ErrKeychainNotSupported)
}
func (x *client) ListKeyChainNamespaces(prefix types.NamespacePrefix) ([]types.Namespace, error) {
	return nil, goerr.Wrap(types.ErrKeychainNotSupported)
}
func (x *client) DeleteKeyChainValue(types.Namespace, types.EnvKey) error {
	return goerr.Wrap(types.ErrKeychainNotSupported)
}
