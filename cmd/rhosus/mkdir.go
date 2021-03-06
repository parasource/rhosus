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
	createDirCmd.SetHelpTemplate(`
Usage:
  rhosus mkdir [path] 

Options:
  -h [--help]			show help information
`)

	rootCmd.AddCommand(createDirCmd)

	createDirCmd.Flags().String("host", "127.0.0.1", "host")
	createDirCmd.Flags().String("port", "5050", "port")
	viper.BindPFlag("host", createDirCmd.Flags().Lookup("host"))
	viper.BindPFlag("port", createDirCmd.Flags().Lookup("port"))
}

var createDirCmd = &cobra.Command{
	Use:   "mkdir",
	Short: "create new directory",
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

		path := args[0]
		res, err := (*client).MakeDir(ctx, &api_pb.MakeDirRequest{
			Path: path,
		})
		if err != nil {
			log.Fatalf("Error from server: %v", err)
		}
		if !res.Success {
			log.Fatalf("Error: %v", res.Err)
		}

		lctx := log.WithField("path", path)
		lctx.Infof("Successfully created a dir")
	},
}
