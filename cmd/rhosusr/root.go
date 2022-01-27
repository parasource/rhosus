package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/registry"
	file_server "github.com/parasource/rhosus/rhosus/server"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"runtime"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs": 0,
	// http file server host and port
	"http_host": "127.0.0.1",
	"http_port": "8000",
	// seconds to wait til force shutdown
	"shutdown_timeout": 30,
	// how many times a file should be replicated
	"replication_factor": 1,
	// block size in bytes
	"block_size": 4096,
}

type DefaultChecker struct {
	flags *pflag.FlagSet
}

func (c *DefaultChecker) checkIfUsingDefault(name string) bool {
	flag := true

	flag = flag && os.Getenv(name) == ""
	//flag = flag && c.flags.Lookup(name) == nil

	return flag
}

var checker *DefaultChecker

func init() {
	rootCmd.Flags().String("http_host", "127.0.0.1", "file server http host")
	rootCmd.Flags().String("http_port", "8000", "file server http port")
	rootCmd.Flags().Int("shutdown_timeout", 30, "node graceful shutdown timeout")
	rootCmd.Flags().Int("replication_factor", 30, "replication factor")
	rootCmd.Flags().Int("block_size", 4096, "block size in bytes")

	viper.BindPFlag("http_host", rootCmd.Flags().Lookup("http_host"))
	viper.BindPFlag("http_port", rootCmd.Flags().Lookup("http_port"))
	viper.BindPFlag("shutdown_timeout", rootCmd.Flags().Lookup("shutdown_timeout"))
	viper.BindPFlag("replication_factor", rootCmd.Flags().Lookup("replication_factor"))
	viper.BindPFlag("block_size", rootCmd.Flags().Lookup("block size in bytes"))

	checker = &DefaultChecker{
		flags: rootCmd.Flags(),
	}
}

var rootCmd = &cobra.Command{
	Use: "rhosusr",
	Run: func(cmd *cobra.Command, args []string) {

		printWelcome()

		for k, v := range configDefaults {
			viper.SetDefault(k, v)
		}

		bindEnvs := []string{
			"http_host", "http_port", "grpc_host", "grpc_port", "redis_host", "redis_port",
			"shutdown_timeout",
			"replication_factor", "block_size",
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

		if checker.checkIfUsingDefault("http_host") || checker.checkIfUsingDefault("http_port") {
			logrus.Warn("file server http address is not set explicitly")
		}

		httpHost := v.GetString("http_host")
		httpPort := v.GetString("http_port")

		conf := registry.Config{
			ServerConfig: file_server.ServerConfig{
				Host:      httpHost,
				Port:      httpPort,
				MaxSizeMb: 0,
				BlockSize: 0,
				PageSize:  0,
			},
		}
		r, err := registry.NewRegistry(conf)
		if err != nil {
			logrus.Fatalf("error creating registry instance: %v", err)
		}

		r.Start()
	},
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus registry")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
