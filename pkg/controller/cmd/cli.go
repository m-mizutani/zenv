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
	var dotEnvFiles cli.StringSlice
	var ignoreErrors cli.StringSlice

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

			&cli.StringSliceFlag{
				Name:        "env-file",
				Usage:       "specify .env file",
				Aliases:     []string{"e"},
				Destination: &dotEnvFiles,
				Value:       cli.NewStringSlice(string(model.DefaultDotEnvFilePath)),
			},

			&cli.StringFlag{
				Name:        "override",
				Usage:       "override .env file",
				Aliases:     []string{"o"},
				Destination: (*string)(&appCfg.OverrideEnvFile),
			},

			&cli.StringSliceFlag{
				Name:        "ignore-error",
				Usage:       "ignore error [env_file_open]",
				Aliases:     []string{"i"},
				Destination: &ignoreErrors,
			},
		},
		Commands: []*cli.Command{
			x.cmdSecret(),
			x.cmdList(),
		},
		Before: func(c *cli.Context) error {
			x.usecase = x.usecase.Clone(usecase.WithConfig(&appCfg))

			appCfg.DotEnvFiles = make([]types.FilePath, len(dotEnvFiles.Value()))
			for i, v := range dotEnvFiles.Value() {
				appCfg.DotEnvFiles[i] = types.FilePath(v)
			}

			appCfg.IgnoreErrors = make(map[types.IgnoreError]struct{})
			for _, v := range ignoreErrors.Value() {
				if !types.IsIgnoreErrorCode(v) {
					return fmt.Errorf("invalid ignore error: %s", v)
				}
				appCfg.IgnoreErrors[types.IgnoreError(v)] = struct{}{}
			}

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
