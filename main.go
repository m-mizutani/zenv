package main

import (
	"context"
	"os"

	"github.com/m-mizutani/zenv/v2/pkg/cli"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func main() {
	if err := cli.Run(context.Background(), os.Args); err != nil {
		// cli.Run is responsible for rendering err to stderr (structured,
		// log-level aware). We just translate it into an exit code.
		os.Exit(model.GetExitCode(err))
	}
}
