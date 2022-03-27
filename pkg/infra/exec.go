package infra

import (
	"bytes"
	"fmt"
	"io"
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

func (x *client) Command(args types.Arguments) (io.Reader, error) {
	argv := args.Strings()

	var cmd *exec.Cmd
	switch len(argv) {
	case 0:
		return nil, goerr.Wrap(types.ErrInnerCommandFailed, "no command")
	case 1:
		cmd = exec.Command(argv[0]) // #nosec
	default:
		cmd = exec.Command(argv[0], argv[1:]...) // #nosec
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return nil, types.ErrInnerCommandFailed.Wrap(err).With("argv", argv)
	}

	return bytes.NewReader(buf.Bytes()), nil
}
