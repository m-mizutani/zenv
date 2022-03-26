//go:build darwin
// +build darwin

package infra

import (
	"strings"

	"github.com/keybase/go-keychain"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

func (x *client) PutKeyChainValues(envVars []*model.EnvVar, ns types.Namespace) error {
	for _, v := range envVars {
		item := keychain.NewItem()
		item.SetSecClass(keychain.SecClassGenericPassword)
		item.SetService(ns.String())
		item.SetAccount(v.Key.String())
		item.SetDescription("zenv")
		item.SetData([]byte(v.Value))
		item.SetAccessible(keychain.AccessibleWhenUnlocked)
		item.SetSynchronizable(keychain.SynchronizableNo)

		err := keychain.AddItem(item)
		if err == keychain.ErrorDuplicateItem {
			// Duplicate
			query := keychain.NewItem()
			query.SetSecClass(keychain.SecClassGenericPassword)
			query.SetService(ns.String())
			query.SetAccount(v.Key.String())
			query.SetMatchLimit(keychain.MatchLimitAll)

			if err := keychain.UpdateItem(query, item); err != nil {
				return goerr.Wrap(err, "Fail to update an existing item")
			}
		} else if err != nil {
			return goerr.Wrap(err, "Fail to add a new keychain item")
		}
	}

	return nil
}

func (x *client) GetKeyChainValues(ns types.Namespace) ([]*model.EnvVar, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(ns.String())
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		if err == keychain.ErrorItemNotFound {
			return nil, types.ErrKeychainNotFound.Wrap(err)
		}
		return nil, goerr.Wrap(err, "Fail to get keychain values")
	}
	if len(results) == 0 {
		return nil, goerr.Wrap(types.ErrKeychainNotFound).With("namespace", ns)
	}

	var envVars []*model.EnvVar
	for _, result := range results {
		q := keychain.NewItem()
		q.SetSecClass(keychain.SecClassGenericPassword)
		q.SetService(ns.String())
		q.SetMatchLimit(keychain.MatchLimitOne)
		q.SetAccount(result.Account)
		q.SetReturnData(true)

		data, err := keychain.QueryItem(q)
		if err != nil {
			return nil, goerr.Wrap(types.ErrKeychainQueryFailed).With("account", result.Account)
		}
		envVars = append(envVars, &model.EnvVar{
			Key:    types.EnvKey(result.Account),
			Value:  types.EnvValue(data[0].Data),
			Secret: true,
		})
	}

	return envVars, nil
}

func (x *client) ListKeyChainNamespaces(prefix types.NamespacePrefix) ([]types.Namespace, error) {
	namespaces := map[types.Namespace]struct{}{}

	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, goerr.Wrap(err, "Fail to get keychain values")
	}
	if len(results) == 0 {
		return nil, goerr.Wrap(types.ErrKeychainNotFound).With("prefix", prefix)
	}

	for _, result := range results {
		if strings.HasPrefix(result.Service, prefix.String()) {
			namespaces[types.Namespace(result.Service)] = struct{}{}
		}
	}

	var resp []types.Namespace
	for ns := range namespaces {
		resp = append(resp, ns)
	}
	return resp, nil
}

func (x *client) DeleteKeyChainValue(ns types.Namespace, key types.EnvKey) error {
	q := keychain.NewItem()
	q.SetSecClass(keychain.SecClassGenericPassword)
	q.SetService(ns.String())
	q.SetAccount(key.String())
	if err := keychain.DeleteItem(q); err != nil {
		if err == keychain.ErrorItemNotFound {
			return types.ErrKeychainNotFound.Wrap(err)
		}
		return err
	}

	return nil
}
