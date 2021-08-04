package controller

import (
	"log"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/interfaces"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/usecase"

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
}

func (x *Controller) CLI(args []string) {
	var cliCfg cliConfig
	var appCfg model.Config

	app := &cli.App{
		Name:  "zenv",
		Usage: "Environment variable manager",
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

			&cli.StringFlag{
				Name:        "config",
				Usage:       "Config file path",
				Aliases:     []string{"c"},
				Destination: &appCfg.ConfigFilePath,
				Value:       filepath.Join(os.Getenv("HOME"), ".zenv"),
			},
		},

		Action: func(c *cli.Context) error {
			x.usecase.SetConfig(&appCfg)

			switch {
			case cliCfg.writeInput.Namespace != "":
				if v := c.Args().Get(0); v != "" {
					cliCfg.writeInput.Key = v
				} else {
					return goerr.Wrap(model.ErrNotEnoughArgument, "key name is required")
				}
				return x.usecase.Write(&cliCfg.writeInput)

			default:
				cliCfg.execInput.Args = c.Args().Slice()
				return x.usecase.Exec(&cliCfg.execInput)
			}
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
