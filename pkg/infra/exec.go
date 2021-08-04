package infra

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
)

func (x *Infrastructure) Exec(vars []*model.EnvVar, args []string) error {
	if len(args) == 0 {
		return model.ErrNotEnoughArgument
	}

	binary, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}

	envvars := os.Environ()
	for _, v := range vars {
		envvars = append(envvars, fmt.Sprintf("%s=%s", v.Key, v.Value))
	}

	if err := syscall.Exec(binary, args, envvars); err != nil {
		return goerr.Wrap(err).With("args", args)
	}

	return nil
}
