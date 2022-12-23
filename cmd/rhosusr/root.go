package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/api"
	"github.com/parasource/rhosus/rhosus/backend"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/registry/cluster"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs": 0,
	// http file server host and port
	"http_host":    "127.0.0.1",
	"http_port":    "8000",
	"db_file_path": "./data.db",

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
	rootCmd.Flags().String("api_addr", "127.0.0.1:8000", "api server address")
	rootCmd.Flags().String("rhosus_path", "./data.db", "rhosus data path")
	rootCmd.Flags().Int("shutdown_timeout", 30, "node graceful shutdown timeout")
	rootCmd.Flags().Int("replication_factor", 30, "replication factor")
	rootCmd.Flags().Int("block_size", 4096, "block size in bytes")

	viper.BindPFlag("cluster_addr", rootCmd.Flags().Lookup("cluster_addr"))
	viper.BindPFlag("rhosus_path", rootCmd.Flags().Lookup("rhosus_path"))
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
			"api_addr", "cluster_addr",
			"shutdown_timeout", "replication_factor", "block_size", "rhosus_path",
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

		shutdownCh := make(chan struct{}, 1)

		conf, err := registryConfig(v)

		r, err := registry.NewRegistry(conf)
		if err != nil {
			logrus.Fatalf("error creating registry instance: %v", err)
		}

		go r.Start()

		apiAddr := v.GetString("api_addr")
		httpApi, err := api.NewApi(r, api.Config{
			Address: apiAddr,
		})
		go httpApi.Run()

		go handleSignals(shutdownCh)
		for {
			select {
			case <-shutdownCh:
				httpApi.Shutdown()
				r.Shutdown()
				return
			}
		}
	},
}

func registryConfig(v *viper.Viper) (registry.Config, error) {
	if checker.checkIfUsingDefault("api_addr") {
		logrus.Warn("api address is not set explicitly")
	}
	if checker.checkIfUsingDefault("cluster_addr") {
		logrus.Warn("cluster address is not set explicitly")
	}

	v4uid, _ := uuid.NewV4()
	id := v4uid.String()

	clusterAddr := v.GetString("cluster_addr")
	rhosusPath := v.GetString("rhosus_path")

	conf := registry.Config{
		ID:         id,
		RhosusPath: rhosusPath,
		Backend: backend.Config{
			Path:          path.Join(rhosusPath, "data"),
			WriteTimeoutS: 10,
			NumWorkers:    5,
		},
		Cluster: cluster.Config{
			WalPath:     path.Join(rhosusPath, "wal"),
			ClusterAddr: clusterAddr,
			ID:          id,
		},
	}

	return conf, nil
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
