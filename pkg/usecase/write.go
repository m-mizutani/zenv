package usecase

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/utils"
)

func (x *Usecase) Write(input *model.WriteSecretInput) error {
	if err := input.Namespace.Validate(); err != nil {
		return err
	}
	if err := input.Key.Validate(); err != nil {
		return err
	}

	value := x.client.Prompt("Value")
	if value == "" {
		utils.Logger.Warn().Msg("No value provided, abort")
		return nil
	}

	envvar := &model.EnvVar{
		Key:    input.Key,
		Value:  types.EnvValue(value),
		Secret: true,
	}
	namespace := input.Namespace.ToNamespace(x.config.KeychainNamespacePrefix)
	if err := x.client.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", input.Key)
	}

	return nil
}

func (x *Usecase) Generate(input *model.GenerateSecretInput) error {
	if err := input.Namespace.Validate(); err != nil {
		return err
	}
	if err := input.Key.Validate(); err != nil {
		return err
	}
	if input.Length < 1 || 65335 < input.Length {
		return goerr.Wrap(types.ErrInvalidArgument, "variable length must be between 1 and 65335")
	}

	value, err := genRandomSecret(uint(input.Length))
	if err != nil {
		return err
	}

	envvar := &model.EnvVar{
		Key:    input.Key,
		Value:  types.EnvValue(value),
		Secret: true,
	}
	namespace := input.Namespace.ToNamespace(x.config.KeychainNamespacePrefix)
	if err := x.client.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", input.Key)
	}

	return nil
}

func genRandomSecret(n uint) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", types.ErrGenerateRandom.Wrap(err)
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}
