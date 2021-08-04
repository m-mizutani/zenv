package main

import (
	"os"

	"github.com/m-mizutani/zenv/pkg/controller"
)

func main() {
	controller.New().CLI(os.Args)
}
