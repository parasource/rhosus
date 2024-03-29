/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package main

import (
	"fmt"
	rhosusnode "github.com/parasource/rhosus/rhosus/node"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"path"
	"runtime"
)

const (
	uuidFileName = "datanode.uuid"
)

var configDefaults = map[string]interface{}{
	"gomaxprocs":       0,
	"service_addr":     "127.0.0.1:5400",
	"rhosus_path":      "/var/lib/rhosus",
	"shutdown_timeout": 30,
}

func init() {
	rootCmd.Flags().String("service_addr", "127.0.0.1:5400", "data node service address")
	rootCmd.Flags().String("etcd_addr", "127.0.0.1:2379", "etcd service discovery address")
	rootCmd.Flags().String("rhosus_path", "/var/lib/rhosus", "rhosus root path")
	rootCmd.Flags().Int("shutdown_timeout", 30, "node shutdown timeout")

	viper.BindPFlag("service_addr", rootCmd.Flags().Lookup("service_addr"))
	viper.BindPFlag("etcd_addr", rootCmd.Flags().Lookup("etcd_addr"))
	viper.BindPFlag("rhosus_path", rootCmd.Flags().Lookup("rhosus_path"))
	viper.BindPFlag("shutdown_timeout", rootCmd.Flags().Lookup("shutdown_timeout"))
}

var rootCmd = &cobra.Command{
	Use: "rhosusd",
	Run: func(cmd *cobra.Command, args []string) {

		printWelcome()

		for k, v := range configDefaults {
			viper.SetDefault(k, v)
		}

		bindEnvs := []string{
			"service_addr", "rhosus_path", "shutdown_timeout", "etcd_addr",
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

		serviceAddr := v.GetString("service_addr")
		etcdAddr := v.GetString("etcd_addr")

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

		nodeId := getId(rhosusPath, true)

		config := rhosusnode.Config{
			ID:          nodeId,
			EtcdAddress: etcdAddr,
			Address:     serviceAddr,
			RhosusPath:  rhosusPath,
		}

		node, _ := rhosusnode.NewNode(config)
		node.Start()
	},
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
			log.Fatal().Err(err).Msg("error opening node uuid file")
		}
		data, err := io.ReadAll(file)
		if err != nil {
			log.Fatal().Err(err).Msg("error reading uuid file")
		}

		id = string(data)
	} else {
		v4uid, _ := uuid.NewV4()
		id = v4uid.String()

		file, err := os.OpenFile(uuidFilePath, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			log.Fatal().Err(err).Msg("error reading uuid file")
		}
		defer file.Close()

		file.Write([]byte(id))
	}

	return id
}

func printWelcome() {
	welcome := "    ____  __  ______  _____ __  _______\n   / __ \\/ / / / __ \\/ ___// / / / ___/\n  / /_/ / /_/ / / / /\\__ \\/ / / /\\__ \\ \n / _, _/ __  / /_/ /___/ / /_/ /___/ / \n/_/ |_/_/ /_/\\____//____/\\____//____/  \n                                       "
	fmt.Println(welcome)

	fmt.Println("\n|------ Rhosus node")
	fmt.Println("|------ Version " + util.Version() + "\n")
}
