package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/api"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs": 0,
	// http file server host and port
	"http_host": "127.0.0.1",
	"http_port": "8000",

	// path for wal
	"wal_path": "wal",
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
	rootCmd.Flags().String("wal_path", "wal", "path for wal")
	rootCmd.Flags().Int("shutdown_timeout", 30, "node graceful shutdown timeout")
	rootCmd.Flags().Int("replication_factor", 30, "replication factor")
	rootCmd.Flags().Int("block_size", 4096, "block size in bytes")

	viper.BindPFlag("http_host", rootCmd.Flags().Lookup("http_host"))
	viper.BindPFlag("http_port", rootCmd.Flags().Lookup("http_port"))
	viper.BindPFlag("wal_path", rootCmd.Flags().Lookup("wal_path"))
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
			"replication_factor", "block_size", "wal_path",
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

		shutdownCh := make(chan struct{}, 1)

		conf := registry.Config{}

		r, err := registry.NewRegistry(conf)
		if err != nil {
			logrus.Fatalf("error creating registry instance: %v", err)
		}

		go r.Start()

		httpHost := v.GetString("http_host")
		httpPort := v.GetString("http_port")

		httpApi, err := api.NewApi(r, api.Config{
			Host: httpHost,
			Port: httpPort,
		})
		go httpApi.Run()

		go handleSignals(shutdownCh)
		for {
			if _, ok := <-shutdownCh; ok {
				httpApi.Shutdown()
				err := r.Shutdown()
				if err != nil {

				}
				return
			}
		}
	},
}

func handleSignals(shutdownCh chan<- struct{}) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)
	for {
		sig := <-sigc
		logrus.Infof("signal received: %v", sig)
		switch sig {
		case syscall.SIGHUP:

		case syscall.SIGINT, os.Interrupt, syscall.SIGTERM:

			pidFile := viper.GetString("pid_file")
			shutdownTimeout := time.Duration(viper.GetInt("shutdown_timeout")) * time.Second

			close(shutdownCh)

			go time.AfterFunc(shutdownTimeout, func() {
				if pidFile != "" {
					os.Remove(pidFile)
				}
				os.Exit(1)
			})
		}
	}
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus registry")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
