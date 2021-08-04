package usecase

import (
	"strings"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/infra"
)

type Usecase struct {
	infra  interfaces.Infra
	config *model.Config
}

func newUsecase() *Usecase {
	return &Usecase{
		infra:  infra.New(),
		config: &model.Config{},
	}
}

func New() interfaces.Usecase {
	return newUsecase()
}

func (x *Usecase) SetConfig(config *model.Config) {
	x.config = config
}

const (
	envVarSeparator = "="
)

func (x *Usecase) Exec(input *model.ExecInput) error {
	envVars := input.EnvVars

	last := 0

ParseLoop:
	for idx, arg := range input.Args {
		switch {
		case strings.Index(arg, envVarSeparator) > 0:
			v := strings.Split(arg, envVarSeparator)
			envVars = append(envVars, &model.EnvVar{
				Key:   v[0],
				Value: strings.Join(v[1:], envVarSeparator),
			})

		case model.IsKeychainNamespace(arg):
			namespace := model.KeychainNamespace(x.config.KeychainNamespacePrefix, arg)
			vars, err := x.infra.GetKeyChainValues(namespace)
			if err != nil {
				return err
			}
			envVars = append(envVars, vars...)

		default:
			break ParseLoop
		}

		last = idx + 1
	}

	args := input.Args[last:]
	if len(args) < 1 {
		return model.ErrNotEnoughArgument
	}

	if err := x.infra.Exec(envVars, args); err != nil {
		return err
	}

	return nil
}

func (x *Usecase) Write(input *model.WriteInput) error {
	if !model.IsKeychainNamespace(input.Namespace) {
		return goerr.Wrap(model.ErrKeychainInvalidNamespace).With("namespace", input.Namespace)
	}

	namespace := model.KeychainNamespace(x.config.KeychainNamespacePrefix, input.Namespace)
	if err := x.infra.PutKeyChainValues([]*model.EnvVar{&input.EnvVar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("!", input.Key)
	}

	return nil
}
