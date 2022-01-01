package main

import (
	"fmt"
	rhosusnode "github.com/parasource/rhosus/rhosus/node"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"runtime"
)

var nodeConfigDefaults = map[string]interface{}{
	"gomaxprocs":       0,
	"registry_host":    "127.0.0.1",
	"registry_port":    "6435",
	"dir":              os.TempDir(),
	"shutdown_timeout": 30,
}

func init() {
	rootCmd.Flags().String("registry_host", "127.0.0.1", "registry grpc node server host")
	rootCmd.Flags().String("registry_port", "6435", "registry grpc node server host")
	rootCmd.Flags().String("dir", os.TempDir(), "node files directory")
	rootCmd.Flags().Int("shutdown_timeout", 30, "node shutdown timeout")

	viper.BindPFlag("registry_host", rootCmd.Flags().Lookup("registry_host"))
	viper.BindPFlag("registry_port", rootCmd.Flags().Lookup("registry_port"))
	viper.BindPFlag("dir", rootCmd.Flags().Lookup("dir"))
	viper.BindPFlag("shutdown_timeout", rootCmd.Flags().Lookup("shutdown_timeout"))
}

var rootCmd = &cobra.Command{
	Use: "rhosusd",
	Run: func(cmd *cobra.Command, args []string) {

		printWelcome()

		for k, v := range nodeConfigDefaults {
			viper.SetDefault(k, v)
		}

		bindEnvs := []string{
			"registry_host", "registry_port", "dir", "shutdown_timeout",
		}
		for _, env := range bindEnvs {
			err := viper.BindEnv(env)
			if err != nil {
				logrus.Fatalf("error binding env variable: %v", err)
			}
		}

		if os.Getenv("GOMAXPROCS") == "" {
			if viper.IsSet("gomaxprocs") && viper.GetInt("gomaxprocs") > 0 {
				runtime.GOMAXPROCS(viper.GetInt("gomaxprocs"))
			} else {
				runtime.GOMAXPROCS(runtime.NumCPU())
			}
		}

		v := viper.GetViper()

		rHost := v.GetString("registry_host")
		rPort := v.GetString("registry_port")

		config := rhosusnode.Config{
			RegistryHost: rHost,
			RegistryPort: rPort,
		}

		node, _ := rhosusnode.NewNode(config)
		node.Start()
	},
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus node")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
