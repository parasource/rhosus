package main

func main() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatalf("failed to execute root command: %v", err)
	}
}
