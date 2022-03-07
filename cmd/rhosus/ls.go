package main

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"time"
)

func init() {
	listCmd.SetHelpTemplate(`
Usage:
  rhosus ls [path] 

Options:
  -h [--help]			show help information
`)

	rootCmd.AddCommand(listCmd)

	listCmd.Flags().String("host", "127.0.0.1", "host")
	listCmd.Flags().String("port", "5050", "port")
	viper.BindPFlag("host", listCmd.Flags().Lookup("host"))
	viper.BindPFlag("port", listCmd.Flags().Lookup("port"))
}

var listCmd = &cobra.Command{
	Use:   "ls",
	Short: "list files and directories",
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
		res, err := (*client).List(ctx, &api_pb.ListRequest{
			Path: path,
		})
		if err != nil {
			log.Fatalf("Error from server: %v", err)
		}
		if res.Error != "" {
			log.Fatalf("Error: %v", res.Error)
		}

		for _, file := range res.List {
			fmt.Printf("%-5s %6d %v %4s \n", file.Type, file.Size_, "0777", file.Name)
		}
	},
}
