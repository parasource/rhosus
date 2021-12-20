package main

import "github.com/sirupsen/logrus"

func main() {
	err := rootCmd.Execute()
	if err != nil {
		logrus.Fatalf("failed to execute root command: %v", err)
	}
}
