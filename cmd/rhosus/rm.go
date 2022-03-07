package main

import (
	"context"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"time"
)

func init() {
	rmCmd.SetHelpTemplate(`
Usage:
  rhosus rm [path]

Options:
  -h [--help]	show help information
  -rf [--recursive]	  remove files recursively
`)

	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().String("host", "127.0.0.1", "host")
	rmCmd.Flags().String("port", "5050", "port")
	viper.BindPFlag("host", rmCmd.Flags().Lookup("host"))
	viper.BindPFlag("port", rmCmd.Flags().Lookup("port"))
}

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove file or directory",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetHandler(cli.Default)
		log.SetLevel(log.DebugLevel)

		if len(args) != 1 {
			log.Error("wrong arguments number")
			cmd.Usage()
			return
		}

		v := viper.GetViper()

		host := v.GetString("host")
		port := v.GetString("port")
		address := net.JoinHostPort(host, port)

		client, err := GetConn(address)
		if err != nil {
			log.Errorf("%v", err)
			log.Fatal("Error connecting to registry. Is it running?")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		res, err := (*client).Remove(ctx, &api_pb.RemoveRequest{
			Path:      args[0],
			Recursive: true,
		})
		if err != nil {
			log.Error(err.Error())
		}

		if res.Success {
			log.Infof("file or directory has been removed")
		}
	},
}
