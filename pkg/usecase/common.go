package usecase

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-shellwords"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

func loadDotEnv(filepath types.FilePath, readAll func(types.FilePath) ([]byte, error)) (types.Arguments, error) {
	raw, err := readAll(filepath)
	if err != nil {
		if os.IsNotExist(err) && filepath == model.DefaultDotEnvFilePath {
			return nil, nil // Ignore not exist error for default file path
		}
		return nil, goerr.Wrap(err).With("filepath", filepath)
	}

	lines := strings.Split(string(raw), "\n")
	var args types.Arguments
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue // skip
		}
		args = append(args, types.Argument(line))
	}

	return args, nil
}

func (x *Usecase) loadEnvVar(arg types.Argument) ([]*model.EnvVar, error) {
	switch {
	case arg.HasEnvVarSeparator():
		key, value := arg.ToEnvVar()
		return []*model.EnvVar{
			{
				Key:   key,
				Value: value,
			},
		}, nil

	case arg.IsKeychainNamespace():
		ns := types.NamespaceSuffix(arg).ToNamespace(x.config.KeychainNamespacePrefix)
		vars, err := x.client.GetKeyChainValues(ns)
		if err != nil {
			if errors.Is(err, types.ErrKeychainNotFound) {
				return nil, goerr.Wrap(err).With("namespace", arg)
			}
			return nil, err
		}
		return vars, nil

	default:
		return nil, nil
	}
}

func (x *Usecase) parseArgs(args types.Arguments) (types.Arguments, []*model.EnvVar, error) {
	var envVars []*model.EnvVar

	for _, dotEnvFile := range x.config.DotEnvFiles {
		loaded, err := loadDotEnv(dotEnvFile, x.client.ReadFile)
		if err != nil {
			return nil, nil, err
		}

		for _, arg := range loaded {
			vars, err := x.loadEnvVar(arg)
			if err != nil {
				return nil, nil, err
			} else if vars == nil {
				return nil, nil, goerr.Wrap(types.ErrInvalidArgumentFormat, "in dotenv file").With("arg", arg).With("file", dotEnvFile)
			}

			envVars = append(envVars, vars...)
		}
	}

	last := 0
	for idx, arg := range args {
		vars, err := x.loadEnvVar(arg)
		if err != nil {
			return nil, nil, err
		} else if vars == nil {
			break
		}
		envVars = append(envVars, vars...)
		last = idx + 1
	}

	for _, v := range envVars {
		switch {
		case v.Value.IsFilePath():
			body, err := x.client.ReadFile(v.Value.ToFilePath())
			if err != nil {
				return nil, nil, goerr.Wrap(err, "failed to open for file loader").With("target", v)
			}
			v.Value = types.EnvValue(body)

		case v.Value.IsInnerCommand():
			cmd := v.Value.ToInnerCommand()
			args, err := shellwords.Parse(cmd)
			if err != nil {
				return nil, nil, goerr.Wrap(err, "can not parse inner command").With("var", v)
			}

			r, err := x.client.Command(types.NewArguments(args))
			if err != nil {
				return nil, nil, goerr.Wrap(err, "failed to execute inner command")
			}
			stdout, err := io.ReadAll(r)
			if err != nil {
				return nil, nil, goerr.Wrap(err, "failed to read inner command stdout")
			}

			v.Value = types.EnvValue(stdout)
		}
	}

	// Remove duplicated env vars. The last one is used.
	unique := make(map[types.EnvKey]*model.EnvVar)
	for _, v := range envVars {
		unique[v.Key] = v
	}
	envVars = make([]*model.EnvVar, 0, len(unique))
	for _, v := range unique {
		envVars = append(envVars, v)
	}

	assigned := make([]*model.EnvVar, len(envVars))
	for i, v := range envVars {
		newVar := *v
		for _, r := range envVars {
			key := "%" + string(r.Key)
			newVar.Value = types.EnvValue(strings.ReplaceAll(string(newVar.Value), key, string(r.Value)))
		}
		assigned[i] = &newVar
	}

	return args[last:], assigned, nil
}
