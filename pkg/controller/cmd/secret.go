package cmd

import (
	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/urfave/cli/v2"
)

func (x *Command) cmdSecretWrite() *cli.Command {
	var writeInput model.WriteSecretInput
	return &cli.Command{
		Name:      "write",
		Usage:     "save a value into Keychain",
		UsageText: "save <namespace> <key>",
		Aliases:   []string{"w"},
		Action: func(c *cli.Context) error {
			if c.NArg() != 2 {
				return goerr.Wrap(types.ErrInvalidArgumentFormat, "write [namespace] [key]")
			}
			writeInput.Namespace = types.NamespaceSuffix(c.Args().Get(0))
			writeInput.Key = types.EnvKey(c.Args().Get(1))
			return x.usecase.WriteSecret(&writeInput)
		},
	}
}

func (x *Command) cmdSecretGenerate() *cli.Command {
	var genInput model.GenerateSecretInput

	return &cli.Command{
		Name:      "generate",
		Usage:     "generate a random value and save it into Keychain",
		UsageText: "generate <namespace> <key>",
		Aliases:   []string{"g", "gen"},
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
	}
}

func (x *Command) cmdSecretList() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "output list of namespaces",
		UsageText: "list",
		Aliases:   []string{"l", "ls"},
		Action: func(c *cli.Context) error {
			return x.usecase.ListNamespaces()
		},
	}
}

func (x *Command) cmdSecretDelete() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "delete a value in Keychain",
		UsageText: "delete <namespace> <key>",
		Aliases:   []string{"d", "del"},
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
	}
}

func (x *Command) cmdSecretExport() *cli.Command {
	var exInput model.ExportSecretInput
	var namespaces cli.StringSlice
	return &cli.Command{
		Name:      "export",
		Usage:     "dump encrypted backup file",
		UsageText: "export",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "namespace",
				Aliases:     []string{"n"},
				Usage:       "Target namespace(s). If empty, all namespace will be exported",
				Destination: &namespaces,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Output file name. '-' means stdout. If empty, temporary file will be created",
				Destination: (*string)(&exInput.Output),
			},
		},
		Action: func(c *cli.Context) error {
			values := namespaces.Value()
			ns := make([]types.NamespaceSuffix, len(values))
			for i := range values {
				ns[i] = types.NamespaceSuffix(values[i])
			}
			exInput.Namespaces = ns
			return x.usecase.ExportSecret(&exInput)
		},
	}
}

func (x *Command) cmdSecretImport() *cli.Command {
	return &cli.Command{
		Name:      "import",
		Usage:     "load encrypted backup file",
		UsageText: "import <backup_file>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return goerr.Wrap(types.ErrInvalidArgumentFormat, "import [backup_file]")
			}

			return x.usecase.ImportSecret(&model.ImportSecretInput{
				Input: types.FilePath(c.Args().Get(0)),
			})
		},
	}
}

func (x *Command) cmdSecret() *cli.Command {

	return &cli.Command{
		Name: "secret",
		Subcommands: []*cli.Command{
			x.cmdSecretWrite(),
			x.cmdSecretGenerate(),
			x.cmdSecretList(),
			x.cmdSecretDelete(),
			x.cmdSecretExport(),
			x.cmdSecretImport(),
		},
	}
}
