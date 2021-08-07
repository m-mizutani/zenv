package controller

import (
	"errors"
	"os"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/m-mizutani/zenv/pkg/utils"

	cli "github.com/urfave/cli/v2"
)

type Controller struct {
	usecase interfaces.Usecase
}

func New() *Controller {
	return &Controller{
		usecase: usecase.New(),
	}
}

type cliConfig struct {
	execInput  model.ExecInput
	writeInput model.WriteInput
	ListMode   bool
}

func (x *Controller) CLI(args []string) {
	var cliCfg cliConfig
	var appCfg model.Config

	app := &cli.App{
		Name:    "zenv",
		Usage:   "Environment variable manager",
		Version: model.AppVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "write",
				Usage:       "Write variables to keychain [-w namespace key]",
				Aliases:     []string{"w"},
				Destination: &cliCfg.writeInput.Namespace,
			},
			&cli.StringFlag{
				Name:        "keychian-prefix",
				Usage:       "Keychain name prefix",
				Aliases:     []string{"k"},
				Destination: &appCfg.KeychainNamespacePrefix,
				Value:       "zenv.",
			},

			&cli.BoolFlag{
				Name:        "list",
				Usage:       "show list of loaded environment variabes",
				Aliases:     []string{"l"},
				Destination: &cliCfg.ListMode,
			},

			&cli.StringFlag{
				Name:        "env-file",
				Usage:       "specify dotenv file",
				Aliases:     []string{"e"},
				Destination: &appCfg.DotEnvFile,
				Value:       model.DefaultDotEnvFilePath,
			},
		},

		Action: func(c *cli.Context) error {
			x.usecase.SetConfig(&appCfg)

			switch {
			case cliCfg.writeInput.Namespace != "":
				cliCfg.writeInput.Args = c.Args().Slice()
				return x.usecase.Write(&cliCfg.writeInput)

			case cliCfg.ListMode:
				return x.usecase.List(&model.ListInput{
					Args: c.Args().Slice(),
				})

			default:
				cliCfg.execInput.Args = c.Args().Slice()
				return x.usecase.Exec(&cliCfg.execInput)
			}
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
