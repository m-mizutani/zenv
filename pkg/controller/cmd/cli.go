package cmd

import (
	"errors"
	"os"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
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
				Destination: &appCfg.KeychainNamespacePrefix,
				Value:       "zenv.",
			},

			&cli.StringFlag{
				Name:        "env-file",
				Usage:       "specify dotenv file",
				Aliases:     []string{"e"},
				Destination: &appCfg.DotEnvFile,
				Value:       model.DefaultDotEnvFilePath,
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
			input.Args = c.Args().Slice()
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
			return x.usecase.List(&model.ListInput{
				Args: c.Args().Slice(),
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
						return goerr.Wrap(model.ErrInvalidArgumentFormat, "write [namespace] [key]")
					}
					writeInput.Namespace = c.Args().Get(0)
					writeInput.Key = c.Args().Get(1)
					return x.usecase.Write(&writeInput)
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
						return goerr.Wrap(model.ErrInvalidArgumentFormat, "generate [namespace] [key]")
					}
					genInput.Namespace = c.Args().Get(0)
					genInput.Key = c.Args().Get(1)
					return x.usecase.Generate(&genInput)
				},
			},
		},
	}
}