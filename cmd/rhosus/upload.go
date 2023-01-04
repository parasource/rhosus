package main

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func init() {
	uploadCmd.SetHelpTemplate(`
Usage:
  rhosus upload [path] [remote path]

Options:
  -h [--help]	show help information
`)

	rootCmd.AddCommand(uploadCmd)
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "upload a file",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			print(arg)
		}
		println()
		log.Infof("file has been uploaded")
	},
}
