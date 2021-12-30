package rhosus_node

import (
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
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
	GrpcServer   *GrpcServer
	EtcdClient   *rhosus_etcd.EtcdClient

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

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Host: "localhost",
		Port: "2379",
	})
	if err != nil {
		logrus.Fatalf("error connecting to etcd: %v", err)
	}
	node.EtcdClient = etcdClient

	grpcServer, err := NewGrpcServer(GrpcServerConfig{
		Host: "localhost",
		Port: "2232",
	}, node)
	if err != nil {
		logrus.Errorf("error creating node grpc server: %v", err)
	}
	node.GrpcServer = grpcServer

	return node, nil
}

func (n *Node) Start() {

	go n.GrpcServer.Run()

	go n.handleSignals()

	err := n.Register()
	if err != nil {
		logrus.Fatalf("can't register node in etcd: %v", err)
	}

	for {
		select {
		case <-n.shutdownCh:
			return
		}
	}

}

func (n *Node) Register() error {
	info := &transmission_pb.NodeInfo{
		Uid: "node1",
		Address: &transmission_pb.NodeInfo_Address{
			Host: "localhost",
			Port: "2232",
		},
		Metrics: &transmission_pb.NodeMetrics{
			Capacity:   10000,
			Remaining:  5000,
			LastUpdate: time.Now().Add(-time.Hour * 24 * 30).Unix(),
		},
		Location: "/dir/1",
	}
	return n.EtcdClient.RegisterNode("node1", info)
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
