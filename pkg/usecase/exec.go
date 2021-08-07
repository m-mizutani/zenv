package usecase

import (
	"strings"

	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func (x *Usecase) Exec(input *model.ExecInput) error {
	args, vars, err := x.parseArgs(input.Args)
	if err != nil {
		return err
	}

	envVars := append(input.EnvVars, vars...)

	if len(args) < 1 {
		return model.ErrNotEnoughArgument
	}

	if err := x.infra.Exec(envVars, args); err != nil {
		return err
	}

	return nil
}

func (x *Usecase) List(input *model.ListInput) error {
	_, vars, err := x.parseArgs(input.Args)
	if err != nil {
		return err
	}

	envVars := append(input.EnvVars, vars...)

	for _, envVar := range envVars {
		value := envVar.Value
		if envVar.Secret {
			value = strings.Repeat("*", len(envVar.Value)) + " (hidden)"
		}

		x.infra.Stdout("%s=%s\n", envVar.Key, value)
	}

	return nil
}
