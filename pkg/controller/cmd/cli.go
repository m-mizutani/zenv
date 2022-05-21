package cmd

import (
	"errors"
	"fmt"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/m-mizutani/zenv/pkg/utils"

	cli "github.com/urfave/cli/v2"
)

type Command struct {
	usecase *usecase.Usecase
}

func New(options ...Option) *Command {
	cmd := &Command{
		usecase: usecase.New(),
	}

	for _, opt := range options {
		opt(cmd)
	}
	return cmd
}

type Option func(ctrl *Command)

func WithUsecase(usecase *usecase.Usecase) Option {
	return func(ctrl *Command) {
		ctrl.usecase = usecase
	}
}

func (x *Command) Run(args []string) error {
	var appCfg model.Config

	app := &cli.App{
		Name:    "zenv",
		Usage:   "Environment variable manager",
		Version: model.AppVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "keychian-prefix",
				Usage:       "Keychain name prefix",
				Aliases:     []string{"k"},
				Destination: (*string)(&appCfg.KeychainNamespacePrefix),
				Value:       "zenv.",
			},

			&cli.StringFlag{
				Name:        "env-file",
				Usage:       "specify dotenv file",
				Aliases:     []string{"e"},
				Destination: (*string)(&appCfg.DotEnvFile),
				Value:       (string)(model.DefaultDotEnvFilePath),
			},
		},
		Commands: []*cli.Command{
			x.cmdSecret(),
			x.cmdList(),
		},
		Before: func(c *cli.Context) error {
			x.usecase = x.usecase.Clone(usecase.WithConfig(&appCfg))
			return nil
		},
		Action: func(c *cli.Context) error {
			var input model.ExecInput
			args := make(types.Arguments, c.NArg())
			for i := range args {
				args[i] = types.Argument(c.Args().Get(i))
			}
			input.Args = args
			return x.usecase.Exec(&input)
		},
	}

	if err := app.Run(args); err != nil {
		ev := utils.Logger.Error()

		var goErr *goerr.Error
		if errors.As(err, &goErr) {
			for k, v := range goErr.Values() {
				ev = ev.Interface(fmt.Sprintf("%v", k), v)
			}
		}
		ev.Msg(err.Error())

		return err
	}

	return nil
}

func (x *Command) cmdList() *cli.Command {
	return &cli.Command{
		Name: "list",
		Action: func(c *cli.Context) error {
			args := make(types.Arguments, c.NArg())
			for i := range args {
				args[i] = types.Argument(c.Args().Get(i))
			}
			return x.usecase.List(&model.ListInput{
				Args: args,
			})
		},
	}
}
