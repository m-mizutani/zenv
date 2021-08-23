package usecase

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/utils"
)

func (x *usecase) Write(input *model.WriteSecretInput) error {
	if err := model.ValidateKeychainNamespace(input.Namespace); err != nil {
		return goerr.Wrap(err).With("namespace", input.Namespace)
	}
	if err := model.ValidateEnvVarKeyName(input.Key); err != nil {
		return goerr.Wrap(err).With("name", input.Key)
	}

	value := x.infra.Prompt("Value")
	if value == "" {
		utils.Logger.Warn().Msg("No value provided, abort")
		return nil
	}

	envvar := &model.EnvVar{
		Key:    input.Key,
		Value:  value,
		Secret: true,
	}
	namespace := model.KeychainNamespace(x.config.KeychainNamespacePrefix, input.Namespace)
	if err := x.infra.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", input.Key)
	}

	return nil
}

func (x *usecase) Generate(input *model.GenerateSecretInput) error {
	if err := model.ValidateKeychainNamespace(input.Namespace); err != nil {
		return goerr.Wrap(err).With("namespace", input.Namespace)
	}
	if err := model.ValidateEnvVarKeyName(input.Key); err != nil {
		return goerr.Wrap(err).With("name", input.Key)
	}
	if input.Length < 1 || 65335 < input.Length {
		return goerr.Wrap(model.ErrInvalidArgument, "variable length must be between 1 and 65335")
	}

	value, err := genRandomSecret(uint(input.Length))
	if err != nil {
		return err
	}

	envvar := &model.EnvVar{
		Key:    input.Key,
		Value:  value,
		Secret: true,
	}
	namespace := model.KeychainNamespace(x.config.KeychainNamespacePrefix, input.Namespace)
	if err := x.infra.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", input.Key)
	}

	return nil
}

func genRandomSecret(n uint) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", model.ErrGenerateRandom.Wrap(err)
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}
