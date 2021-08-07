package usecase

import (
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/utils"
)

func (x *Usecase) Write(input *model.WriteInput) error {
	if err := model.ValidateKeychainNamespace(input.Namespace); err != nil {
		return goerr.Wrap(err).With("namespace", input.Namespace)
	}

	if len(input.Args) != 1 {
		return goerr.Wrap(model.ErrNotEnoughArgument, "Only 1 key name of environment variable is required").With("args", input.Args)
	}
	key := input.Args[0]

	value := x.infra.Prompt("Value")
	if value == "" {
		utils.Logger.Warn().Msg("No value provided, abort")
		return nil
	}

	envvar := &model.EnvVar{
		Key:    key,
		Value:  value,
		Secret: true,
	}
	namespace := model.KeychainNamespace(x.config.KeychainNamespacePrefix, input.Namespace)
	if err := x.infra.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", key)
	}

	return nil
}
