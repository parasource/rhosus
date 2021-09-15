package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"parasource/rhosus/src/logging"
	rhosusnode "parasource/rhosus/src/node"
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
	startCmd.Flags().StringVarP(&nodeName, "name", "n", "default", "node name")
	startCmd.Flags().StringVarP(&nodeAddress, "address", "a", "127.0.0.1:3617", "node address")
	startCmd.Flags().IntVarP(&nodeTimeout, "timeout", "t", 30, "connection idle seconds")
	startCmd.Flags().StringVarP(&nodeFolders, "dir", "d", os.TempDir(), "directories to store data files. dir[,dir]")

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use: "start",
	Run: runNode,
}

func runNode(cmd *cobra.Command, args []string) {
	config := rhosusnode.Config{
		Name:    nodeName,
		Address: nodeAddress,
		Timeout: time.Second * time.Duration(nodeTimeout),
		Logger:  logging.NewLogHandler(),
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
