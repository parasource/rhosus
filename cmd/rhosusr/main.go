package main

import "log"

func main() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("error running root command: %v", err)
	}
}
