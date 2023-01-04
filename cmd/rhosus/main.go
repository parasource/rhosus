package main

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
)

func main() {
	log.SetHandler(cli.Default)
	log.SetLevel(log.DebugLevel)

	err := Execute()
	if err != nil {
		log.Fatalf("Failed to start execute command: %v", err)
	}
}
