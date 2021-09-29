package usecase

import (
	"errors"
	"os"
	"strings"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

const (
	envVarSeparator  = "="
	envVarFileLoader = "@"
)

func loadDotEnv(filepath string, readAll func(string) ([]byte, error)) ([]string, error) {
	raw, err := readAll(filepath)
	if err != nil {
		if os.IsNotExist(err) && filepath == model.DefaultDotEnvFilePath {
			return nil, nil // Ignore not exist error for default file path
		}
		return nil, goerr.Wrap(err).With("filepath", filepath)
	}

	lines := strings.Split(string(raw), "\n")
	var args []string
	for _, line := range lines {
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue // skip
		}
		args = append(args, line)
	}

	return args, nil
}

func (x *usecase) loadEnvVar(arg string) ([]*model.EnvVar, error) {
	switch {
	case strings.Index(arg, envVarSeparator) > 0:
		v := strings.Split(arg, envVarSeparator)
		return []*model.EnvVar{
			{
				Key:   v[0],
				Value: strings.Join(v[1:], envVarSeparator),
			},
		}, nil

	case model.ValidateKeychainNamespace(arg) == nil:
		namespace := model.KeychainNamespace(x.config.KeychainNamespacePrefix, arg)
		vars, err := x.infra.GetKeyChainValues(namespace)
		if err != nil {
			if errors.Is(err, model.ErrKeychainNotFound) {
				return nil, goerr.Wrap(err).With("namespace", arg)
			}
			return nil, err
		}
		return vars, nil

	default:
		return nil, nil
	}
}

func (x *usecase) parseArgs(args []string) ([]string, []*model.EnvVar, error) {
	var envVars []*model.EnvVar

	if x.config.DotEnvFile != "" {
		loaded, err := loadDotEnv(x.config.DotEnvFile, x.infra.ReadFile)
		if err != nil {
			return nil, nil, err
		}

		for _, arg := range loaded {
			vars, err := x.loadEnvVar(arg)
			if err != nil {
				return nil, nil, err
			} else if vars == nil {
				return nil, nil, goerr.Wrap(model.ErrInvalidArgumentFormat, "in dotenv file").With("arg", arg).With("file", x.config.DotEnvFile)
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
		if strings.HasPrefix(v.Value, envVarFileLoader) {
			body, err := x.infra.ReadFile(strings.TrimPrefix(v.Value, envVarFileLoader))
			if err != nil {
				return nil, nil, goerr.Wrap(err, "failed to open for file loader").With("target", v)
			}
			v.Value = string(body)
		}
	}

	return args[last:], envVars, nil
}
