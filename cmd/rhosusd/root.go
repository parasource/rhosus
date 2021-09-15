package rhosusd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "rhosusd",
	Run: func(cmd *cobra.Command, args []string) {
		println("starting rhosus daemon")
	},
}

func Execute() error {
	return rootCmd.Execute()
}
