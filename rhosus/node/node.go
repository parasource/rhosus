package rhosus_node

import (
	"context"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"net"
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

	mu         sync.RWMutex
	shutdownCh chan struct{}
	readyCh    chan struct{}
}

func NewNode(config Config) *Node {
	return &Node{
		Config: config,

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}),
	}
}

func (n *Node) Start() {

	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-n.shutdownCh:
				return
			case <-ticker.C:

				address := net.JoinHostPort(n.Config.RegistryHost, n.Config.RegistryPort)

				conn, err := grpc.Dial(address, grpc.WithInsecure())
				if err != nil {
					logrus.Errorf("error dialing node server: %v", err)
					conn.Close()
					continue
				}

				client := node_pb.NewNodeServiceClient(conn)

				_, err = client.Ping(context.Background(), &node_pb.PingRequest{})
				if err != nil {
					logrus.Errorf("error sending ping commmand: %v", err)
					conn.Close()
					continue
				}

				logrus.Infof("PONG")

				conn.Close()
			}
		}
	}()

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
