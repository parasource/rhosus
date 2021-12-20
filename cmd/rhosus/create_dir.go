package main

import "github.com/spf13/cobra"

func init() {
	createDirCmd.SetHelpTemplate(`
Usage:
  rhosus dir create [path] 

Options:
  -h [--help]			show help information
`)

	rootCmd.AddCommand(createDirCmd)
}

var createDirCmd = &cobra.Command{
	Use:   "dir create",
	Short: "create new directory",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			println(arg)
		}
	},
}
