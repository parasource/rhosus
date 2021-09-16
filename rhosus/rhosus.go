package main

import (
	"log"
	"parasource/rhosus/cmd/rhosus"
)

func main() {
	err := rhosus.Execute()
	if err != nil {
		log.Fatalf("error running root command: %v", err)
	}
}
