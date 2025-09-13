package main

import (
	"context"
	"os"

	"github.com/m-mizutani/zenv/v2/pkg/cli"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func main() {
	if err := cli.Run(context.Background(), os.Args); err != nil {
		// Print error message to stderr
		// Use goerr's detailed formatting if available
		_, _ = os.Stderr.WriteString("zenv: " + err.Error() + "\n")
		exitCode := model.GetExitCode(err)
		os.Exit(exitCode)
	}
}
