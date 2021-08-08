package usecase

import (
	"sort"
	"strings"

	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func replaceArguments(envVars []*model.EnvVar, args []string) []string {
	// Sort by descending order of Key length
	vars := envVars[:]
	sort.Slice(vars, func(i, j int) bool {
		return len(vars[i].Key) > len(envVars[j].Key)
	})

	replaced := args[:]

	for _, v := range vars {
		key := "%" + v.Key
		for i := range replaced {
			replaced[i] = strings.Replace(replaced[i], key, v.Value, -1)
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
		return model.ErrNotEnoughArgument
	}

	newArgs := replaceArguments(envVars, args)

	if err := x.infra.Exec(envVars, newArgs); err != nil {
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
