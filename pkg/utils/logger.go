package utils

import (
	"os"

	"github.com/rs/zerolog"
)

var Logger zerolog.Logger

func initLogger() {
	Logger = zerolog.New(os.Stderr).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
