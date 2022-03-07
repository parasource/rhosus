package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/util"
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
		printWelcome()
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus CLI")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
