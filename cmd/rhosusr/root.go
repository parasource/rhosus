package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"runtime"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs":       0,
	"http_host":        "127.0.0.1",
	"http_port":        "8000",
	"grpc_host":        "127.0.0.1",
	"grpc_port":        "8080",
	"shutdown_timeout": 30,
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

	viper.BindPFlag("http_host", rootCmd.Flags().Lookup("http_host"))
	viper.BindPFlag("http_port", rootCmd.Flags().Lookup("http_port"))
	viper.BindPFlag("shutdown_timeout", rootCmd.Flags().Lookup("shutdown_timeout"))

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
		if checker.checkIfUsingDefault("grpc_host") || checker.checkIfUsingDefault("grpc_port") {
			logrus.Warn("file server grpc address is not set explicitly")
		}

		httpHost := v.GetString("http_host")
		httpPort := v.GetString("http_port")

		conf := registry.RegistryConfig{
			HttpHost: httpHost,
			HttpPort: httpPort,
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
