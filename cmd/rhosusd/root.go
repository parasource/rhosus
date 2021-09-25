package main

import (
	"github.com/parasource/rhosus/rhosus/logging"
	rhosusnode "github.com/parasource/rhosus/rhosus/node"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	nodeName    string
	nodeAddress string
	nodeTimeout int
	nodeFolders string

	log = logrus.New()
)

func init() {
	rootCmd.Flags().StringVarP(&nodeName, "name", "n", "default", "node name")
	rootCmd.Flags().StringVarP(&nodeAddress, "address", "a", "127.0.0.1:3617", "node address")
	rootCmd.Flags().IntVarP(&nodeTimeout, "timeout", "t", 30, "connection idle seconds")
	rootCmd.Flags().StringVarP(&nodeFolders, "dir", "d", os.TempDir(), "directories to store storage_object files. dir[,dir]")
}

var rootCmd = &cobra.Command{
	Use: "rhosusd",
	Run: startupRhosusServer,
}

func startupRhosusServer(cmd *cobra.Command, args []string) {
	config := rhosusnode.Config{
		Name:    nodeName,
		Address: nodeAddress,
		Timeout: time.Second * time.Duration(nodeTimeout),
		Logger:  rlog.NewLogHandler(),
	}

	node := rhosusnode.NewNode(config)
	err := node.Start()
	if err != nil {
		log.Fatalf("error starting node: %v", err)
	}

	handleSignals(node)
}

func handleSignals(node *rhosusnode.Node) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)

	for {
		sig := <-sigc
		log.Infof("signal received: %v", sig)
		switch sig {
		case syscall.SIGHUP:

		case syscall.SIGINT, os.Interrupt, syscall.SIGTERM:
			log.Infof("shutting node down")

			node.Shutdown()

			os.Exit(0)
		}
	}
}
