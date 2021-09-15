package main

import (
	"log"
	"parasource/rhosus/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatalf("error running root command: %v", err)
	}
}
