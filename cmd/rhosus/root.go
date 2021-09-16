package rhosus

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "rhosus",
	Run: func(cmd *cobra.Command, args []string) {
		println("testing root command")
		println(nodeAddress)
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}
