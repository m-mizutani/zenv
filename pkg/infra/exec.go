package infra

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

func (x *client) Exec(vars []*model.EnvVar, args types.Arguments) error {
	if len(args) == 0 {
		return types.ErrNotEnoughArgument
	}

	binary, err := exec.LookPath(args[0].String())
	if err != nil {
		return err
	}

	envvars := os.Environ()
	for _, v := range vars {
		envvars = append(envvars, fmt.Sprintf("%s=%s", v.Key, v.Value))
	}

	/* #nosec */
	if err := syscall.Exec(binary, args.Strings(), envvars); err != nil {
		return goerr.Wrap(err).With("args", args)
	}

	return nil
}
