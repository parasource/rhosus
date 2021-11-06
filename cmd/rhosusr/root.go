package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"runtime"
)

var (
	httpAddress string
	rpcAddress  string
)

var configDefaults = map[string]interface{}{
	"gomaxprocs": 0,
	"redis_host": "127.0.0.1",
	"redis_port": "6379",
}

func init() {
	rootCmd.Flags().StringVarP(&httpAddress, "http-address", "o", "127.0.0.1:8000", "registry http address")
	rootCmd.Flags().StringVarP(&rpcAddress, "rpc-address", "d", "127.0.0.1:8080", "registry rpc address")
	rootCmd.Flags().StringVarP(&rpcAddress, "redis-host", "", "127.0.0.1", "redis hosts")
	rootCmd.Flags().StringVarP(&rpcAddress, "redis-port", "", "6379", "redis ports")
}

var rootCmd = &cobra.Command{
	Use: "rhosusr",
	Run: func(cmd *cobra.Command, args []string) {

		printWelcome()

		for k, v := range configDefaults {
			viper.SetDefault(k, v)
		}

		bindEnvs := []string{
			"redis_host", "redis_port",
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

		conf := registry.RegistryConfig{
			HttpAddress: httpAddress,
			RpcAddress:  rpcAddress,
		}
		r, err := registry.NewRegistry(conf)
		if err != nil {
			logrus.Fatalf("error creating registry instance: %v", err)
		}

		r.Run()
	},
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus registry")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
