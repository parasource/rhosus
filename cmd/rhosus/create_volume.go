package main

import "github.com/spf13/cobra"

func init() {
	cvCmd.SetHelpTemplate(`
Usage:
  rhosus volume create [name] [size]

Options:
  -h [--help]			show help information
`)

	rootCmd.AddCommand(cvCmd)
}

var cvCmd = &cobra.Command{
	Use:   "volume create",
	Short: "create new volume",
	Run: func(cmd *cobra.Command, args []string) {
		println(args)
	},
}
