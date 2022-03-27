package usecase

import (
	"sort"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

func replaceArguments(envVars []*model.EnvVar, args types.Arguments) types.Arguments {
	// Sort by descending order of Key length
	vars := make([]*model.EnvVar, len(envVars))
	copy(vars, envVars)

	sort.Slice(vars, func(i, j int) bool {
		return len(vars[i].Key) > len(envVars[j].Key)
	})

	replaced := args[:]

	for _, v := range vars {
		key := "%" + v.Key
		for i := range replaced {
			replaced[i] = args[i].ReplaceAll(key, v.Value)
		}
	}

	return replaced
}

func (x *Usecase) Exec(input *model.ExecInput) error {
	args, vars, err := x.parseArgs(input.Args)
	if err != nil {
		return err
	}

	envVars := append(input.EnvVars, vars...)

	if len(args) < 1 {
		return types.ErrNotEnoughArgument
	}

	newArgs := replaceArguments(envVars, args)

	if err := x.client.Exec(envVars, newArgs); err != nil {
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
			value = value.ToHiddenValue()
		}

		x.client.Stdout("%s=%s\n", envVar.Key, value)
	}

	return nil
}
