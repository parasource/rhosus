package rhosusd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use: "start",
	Run: func(cmd *cobra.Command, args []string) {
		println("starting rhosus daemon server")
	},
}
