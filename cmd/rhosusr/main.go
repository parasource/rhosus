package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal().Err(err).Msg("error executing root command")
	}
}
