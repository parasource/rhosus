package rhosusd

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use: "status",
	Run: func(cmd *cobra.Command, args []string) {
		println("rhosus daemon status command")
	},
}
