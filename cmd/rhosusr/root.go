package main

import (
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	httpAddress string
	rpcAddress  string
)

func init() {
	rootCmd.Flags().StringVarP(&httpAddress, "http-address", "o", "127.0.0.1:8000", "registry http address")
	rootCmd.Flags().StringVarP(&rpcAddress, "rpc-address", "d", "127.0.0.1:8080", "registry rpc address")
}

var rootCmd = &cobra.Command{
	Use: "rhosusr",
	Run: func(cmd *cobra.Command, args []string) {

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
