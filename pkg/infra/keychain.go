package infra

import (
	"github.com/keybase/go-keychain"
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func (x *Infrastructure) PutKeyChainValues(envVars []*model.EnvVar, namespace string) error {
	for _, v := range envVars {
		item := keychain.NewItem()
		item.SetSecClass(keychain.SecClassGenericPassword)
		item.SetService(namespace)
		item.SetAccount(v.Key)
		item.SetDescription("altenv")
		item.SetData([]byte(v.Value))
		item.SetAccessible(keychain.AccessibleWhenUnlocked)
		item.SetSynchronizable(keychain.SynchronizableNo)

		err := keychain.AddItem(item)
		if err == keychain.ErrorDuplicateItem {
			// Duplicate
			query := keychain.NewItem()
			query.SetSecClass(keychain.SecClassGenericPassword)
			query.SetService(namespace)
			query.SetAccount(v.Key)
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

func (x *Infrastructure) GetKeyChainValues(namespace string) ([]*model.EnvVar, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(namespace)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, goerr.Wrap(err, "Fail to get keychain values")
	}
	if len(results) == 0 {
		return nil, goerr.Wrap(model.ErrKeychainNotFound).With("namespace", namespace)
	}

	var envVars []*model.EnvVar
	for _, result := range results {
		q := keychain.NewItem()
		q.SetSecClass(keychain.SecClassGenericPassword)
		q.SetService(namespace)
		q.SetMatchLimit(keychain.MatchLimitOne)
		q.SetAccount(result.Account)
		q.SetReturnData(true)

		data, err := keychain.QueryItem(q)
		if err != nil {
			return nil, goerr.Wrap(model.ErrKeychainQueryFailed).With("account", result.Account)
		}
		envVars = append(envVars, &model.EnvVar{
			Key:   result.Account,
			Value: string(data[0].Data),
		})
	}

	return envVars, nil
}
