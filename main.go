package main

import (
	"os"

	"github.com/m-mizutani/zenv/pkg/controller/cmd"
)

func main() {
	cmd.New().Run(os.Args) // #nosec, error should be handled in Run()
}
