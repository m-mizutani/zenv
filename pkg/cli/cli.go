package cli

import (
	"context"

	"github.com/m-mizutani/zenv/pkg/executor"
	"github.com/m-mizutani/zenv/pkg/loader"
	"github.com/m-mizutani/zenv/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string) error {
	var envFiles []string
	var tomlFiles []string

	app := &cli.Command{
		Name:  "zenv",
		Usage: "Environment variable loader and command executor",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "env",
				Aliases:     []string{"e"},
				Usage:       "Load environment variables from .env file",
				Destination: &envFiles,
			},
			&cli.StringSliceFlag{
				Name:        "toml",
				Aliases:     []string{"t"},
				Usage:       "Load environment variables from .toml file",
				Destination: &tomlFiles,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var loaders []loader.LoadFunc

			// Add .env files specified by -e option
			for _, envFile := range envFiles {
				loaders = append(loaders, loader.NewDotEnvLoader(envFile))
			}

			// Add default .env file if no -e option specified
			if len(envFiles) == 0 {
				loaders = append(loaders, loader.NewDotEnvLoader(".env"))
			}

			// Add .toml files specified by -t option
			for _, tomlFile := range tomlFiles {
				loaders = append(loaders, loader.NewTOMLLoader(tomlFile))
			}

			// Add default .env.toml file if no -t option specified
			if len(tomlFiles) == 0 {
				loaders = append(loaders, loader.NewTOMLLoader(".env.toml"))
			}

			// Create executor and usecase
			exec := executor.NewDefaultExecutor()
			uc := usecase.NewUseCase(loaders, exec)

			// Get command arguments (excluding program name and flags)
			args := cmd.Args().Slice()

			// If no command specified, force list mode
			if len(args) == 0 {
				args = []string{} // Force empty args to show environment variables
			}

			return uc.Run(ctx, args)
		},
	}

	return app.Run(ctx, args)
}
