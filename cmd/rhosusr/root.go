/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package main

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/api"
	"github.com/parasource/rhosus/rhosus/auth"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/registry/cluster"
	"github.com/parasource/rhosus/rhosus/registry/storage"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"
)

const (
	uuidFileName = "registry.uuid"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs": 0,
	// http file server host and port
	"api_addr":     "127.0.0.1:8000",
	"cluster_addr": "127.0.0.1:8100",
	"etcd_addr":    "127.0.0.1:2379",
	"rhosus_path":  "/var/lib/rhosus",

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
	rootCmd.Flags().String("cluster_addr", "127.0.0.1:8100", "cluster server address")
	rootCmd.Flags().String("etcd_addr", "127.0.0.1:2379", "etcd service discovery address")
	rootCmd.Flags().String("rhosus_path", "/var/lib/rhosus", "rhosus data path")
	rootCmd.Flags().Int("shutdown_timeout", 30, "node graceful shutdown timeout")
	rootCmd.Flags().Int("replication_factor", 30, "replication factor")
	rootCmd.Flags().Int("block_size", 4096, "block size in bytes")

	viper.BindPFlag("api_addr", rootCmd.Flags().Lookup("cluster_addr"))
	viper.BindPFlag("cluster_addr", rootCmd.Flags().Lookup("cluster_addr"))
	viper.BindPFlag("etcd_addr", rootCmd.Flags().Lookup("etcd_addr"))
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
			"api_addr", "cluster_addr", "etcd_addr", "rhosus_path",
			"shutdown_timeout", "replication_factor", "block_size",
		}
		for _, env := range bindEnvs {
			err := viper.BindEnv(env)
			if err != nil {
				log.Fatal().Err(err).Msg("error binding env variable")
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

		rhosusPath := v.GetString("rhosus_path")

		// If rhosus home directory does not exist already,
		// we create a new one
		if ok := util.FileExists(rhosusPath); !ok {
			err := os.Mkdir(rhosusPath, 0755)
			if err != nil {
				log.Fatal().Err(err).Msg("error creating rhosus home directory")
			}
		}
		if err := util.TestDirWritable(rhosusPath); err != nil {
			log.Fatal().Err(err).Msg("rhosus home directory is not writable")
		}

		s, err := storage.NewStorage(storage.Config{
			Path:          path.Join(rhosusPath, "registry"),
			WriteTimeoutS: 10,
			NumWorkers:    5,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("error creating storage")
		}
		roleManager, err := auth.NewRoleManager(s)
		if err != nil {
			log.Fatal().Err(err).Msg("error creating role manager")
		}
		tokenManager, err := auth.NewTokenStore(s)
		if err != nil {
			log.Fatal().Err(err).Msg("error creating token manager")
		}

		authMethods := map[string]auth.Authenticator{
			"credentials": auth.NewCredentialsAuth(roleManager, tokenManager),
		}

		conf, err := registryConfig(v, s)

		r, err := registry.NewRegistry(conf)
		if err != nil {
			log.Fatal().Err(err).Msg("error creating registry instance")
		}

		go r.Start()

		apiAddr := v.GetString("api_addr")
		httpApi, err := api.NewApi(r, tokenManager, api.Config{
			Address:     apiAddr,
			AuthMethods: authMethods,
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

func registryConfig(v *viper.Viper, s *storage.Storage) (registry.Config, error) {
	if os.Getenv("API_ADDR") == "" {
		log.Warn().Msg("API_ADDR is not set explicitly, falling back to default")
	}
	if os.Getenv("CLUSTER_ADDR") == "" {
		log.Warn().Msg("CLUSTER_ADDR is not set explicitly, falling back to default")
	}

	clusterAddr := v.GetString("cluster_addr")
	rhosusPath := v.GetString("rhosus_path")
	etcdAddr := v.GetString("etcd_addr")

	// Generating id for Registry
	registryId := getId(rhosusPath, true)

	conf := registry.Config{
		ID:         registryId,
		RhosusPath: rhosusPath,
		EtcdAddr:   etcdAddr,
		Storage:    s,
		Cluster: cluster.Config{
			WalPath:     path.Join(rhosusPath, "wal"),
			ClusterAddr: clusterAddr,
			ID:          registryId,
		},
	}

	return conf, nil
}

func getId(rhosusPath string, persistent bool) string {
	var id string

	if !persistent {
		v4id, _ := uuid.NewV4()
		return v4id.String()
	}

	uuidFilePath := path.Join(rhosusPath, uuidFileName)

	// since we are just testing, we don't need that yet
	if util.FileExists(uuidFilePath) {
		file, err := os.OpenFile(uuidFilePath, os.O_RDONLY, 0666)
		defer file.Close()

		if err != nil {
			log.Fatal().Err(err).Str("path", uuidFilePath).Msg("error opening node uuid file")
		}
		data, err := io.ReadAll(file)
		if err != nil {
			log.Fatal().Err(err).Str("path", uuidFilePath).Msg("error reading node uuid file")
		}

		id = string(data)
	} else {
		v4uid, _ := uuid.NewV4()
		id = v4uid.String()

		file, err := os.OpenFile(uuidFilePath, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			log.Fatal().Err(err).Msg("error opening node uuid file")
		}
		defer file.Close()
		file.Write([]byte(id))
	}

	return id
}

func handleSignals(shutdownCh chan<- struct{}) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)
	for {
		sig := <-sigc
		log.Info().Str("signal", sig.String()).Msg("system signal received")
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
