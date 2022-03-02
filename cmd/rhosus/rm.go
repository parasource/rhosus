package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	uploadCmd.SetHelpTemplate(`
Usage:
  rhosus rm [path]

Options:
  -h [--help]	show help information
  -rf [--recursive]	  remove files recursively
`)

	rootCmd.AddCommand(rmCmd)
}

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove file or directory",
	Run: func(cmd *cobra.Command, args []string) {
		for _, arg := range args {
			print(arg)
		}
		println()
		logrus.Infof("file has been removed")
	},
}
