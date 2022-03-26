package cmd

import (
	"errors"
	"os"

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

func (x *Command) Run(args []string) {
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

	if err := app.Run(os.Args); err != nil {
		ev := utils.Logger.Error()

		var goErr *goerr.Error
		if errors.As(err, &goErr) {
			for k, v := range goErr.Values() {
				ev = ev.Interface(k, v)
			}
		}
		ev.Msg(err.Error())
	}
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

func (x *Command) cmdSecret() *cli.Command {
	var genInput model.GenerateSecretInput
	var writeInput model.WriteSecretInput

	return &cli.Command{
		Name: "secret",
		Subcommands: []*cli.Command{
			{
				Name:    "write",
				Aliases: []string{"w"},
				Action: func(c *cli.Context) error {
					if c.NArg() != 2 {
						return goerr.Wrap(types.ErrInvalidArgumentFormat, "write [namespace] [key]")
					}
					writeInput.Namespace = types.NamespaceSuffix(c.Args().Get(0))
					writeInput.Key = types.EnvKey(c.Args().Get(1))
					return x.usecase.WriteSecret(&writeInput)
				},
			},
			{
				Name:    "generate",
				Aliases: []string{"g", "gen"},
				Flags: []cli.Flag{
					&cli.Int64Flag{
						Name:        "length",
						Aliases:     []string{"n"},
						Usage:       "variable length",
						Destination: &genInput.Length,
						Value:       32,
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() != 2 {
						return goerr.Wrap(types.ErrInvalidArgumentFormat, "generate [namespace] [key]")
					}
					genInput.Namespace = types.NamespaceSuffix(c.Args().Get(0))
					genInput.Key = types.EnvKey(c.Args().Get(1))
					return x.usecase.GenerateSecret(&genInput)
				},
			},
			{
				Name:    "list",
				Aliases: []string{"l", "ls"},
				Action: func(c *cli.Context) error {
					return x.usecase.ListNamespaces()
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"d", "del"},
				Action: func(c *cli.Context) error {
					if c.NArg() != 2 {
						return goerr.Wrap(types.ErrInvalidArgumentFormat, "delete [namespace] [key]")
					}
					delInput := &model.DeleteSecretInput{
						Namespace: types.NamespaceSuffix(c.Args().Get(0)),
						Key:       types.EnvKey(c.Args().Get(1)),
					}
					return x.usecase.DeleteSecret(delInput)
				},
			},
		},
	}
}
