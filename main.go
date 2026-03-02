package main

import (
	"context"
	"fmt"
	"os"

	"github.com/m-mizutani/zenv/v2/pkg/cli"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func main() {
	if err := cli.Run(context.Background(), os.Args); err != nil {
		exitCode := model.GetExitCode(err)
		if !model.IsExecutorError(err) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(exitCode)
	}
}
