package main

import "log"

func main() {
	err := Execute()
	if err != nil {
		log.Fatal("Failed to start root command", err)
	}
}
