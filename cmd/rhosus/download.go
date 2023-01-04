package main

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func init() {
	downloadCmd.SetHelpTemplate(`
Usage:
  rhosus download [path] [remote path]

Options:
  -h [--help]			show help information
`)

	rootCmd.AddCommand(downloadCmd)
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download a file",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			print(arg)
		}
		println()
		log.Infof("file has been downloaded")
	},
}
