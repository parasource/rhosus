package main

import (
	"github.com/spf13/cobra"
	"os"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs":       0,
	"host":             "127.0.0.1",
	"port":             "6435",
	"dir":              os.TempDir(),
	"shutdown_timeout": 30,
}

func init() {
	rootCmd.PersistentFlags().String("host", "127.0.0.1", "registry grpc host")
	rootCmd.PersistentFlags().String("port", "6435", "registry grpc port")
}

var rootCmd = &cobra.Command{
	Use: "rhosus",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}
