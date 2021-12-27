package rhosus_node

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Name         string
	RegistryHost string
	RegistryPort string
	Timeout      time.Duration
	Dir          []string
}

type Node struct {
	Config Config

	mu sync.RWMutex

	StatsManager *StatsManager

	shutdownCh chan struct{}
	readyCh    chan struct{}
}

func NewNode(config Config) (*Node, error) {

	node := &Node{
		Config: config,

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}),
	}

	statsManager := NewStatsManager(node)
	node.StatsManager = statsManager

	return node, nil
}

//func (n *Node) GetNodeServerClient() (node_pb.NodeServiceClient, *grpc.ClientConn, error) {
//	address := net.JoinHostPort(n.Config.RegistryHost, n.Config.RegistryPort)
//
//	conn, err := grpc.Dial(address, grpc.WithInsecure())
//	if err != nil {
//		conn.Close()
//		return nil, err
//	}
//
//	client := node_pb.NewNodeServiceClient(conn)
//
//	return client, conn
//}

func (n *Node) Start() {

	go n.StatsManager.Run()

	go n.handleSignals()

	for {
		select {
		case <-n.shutdownCh:
			return
		}
	}

}

func (n *Node) NotifyShutdown() <-chan struct{} {
	return n.shutdownCh
}

func (n *Node) Shutdown() {
	n.shutdownCh <- struct{}{}
}

func (n *Node) handleSignals() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)
	for {
		sig := <-sigc
		logrus.Infof("signal received: %v", sig)
		switch sig {
		case syscall.SIGHUP:

		case syscall.SIGINT, os.Interrupt, syscall.SIGTERM:
			logrus.Infof("shutting down registry")
			pidFile := viper.GetString("pid_file")
			shutdownTimeout := time.Duration(viper.GetInt("shutdown_timeout")) * time.Second
			go time.AfterFunc(shutdownTimeout, func() {
				if pidFile != "" {
					os.Remove(pidFile)
				}
				os.Exit(1)
			})

			n.shutdownCh <- struct{}{}

			if pidFile != "" {
				err := os.Remove(pidFile)
				if err != nil {
					logrus.Errorf("error removing pid file: %v", err)
				}
			}
			time.Sleep(time.Second)
			os.Exit(0)
		}
	}

}
