package rhosus_node

import (
	"fmt"
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	"github.com/parasource/rhosus/rhosus/node/profiler"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
	"github.com/parasource/rhosus/rhosus/util"
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
	Name   string
	Config Config

	mu sync.RWMutex

	StatsManager *StatsManager
	Profiler     *profiler.Profiler
	GrpcServer   *GrpcServer
	EtcdClient   *rhosus_etcd.EtcdClient

	shutdownCh chan struct{}
	readyCh    chan struct{}
}

func NewNode(config Config) (*Node, error) {

	node := &Node{
		Name:   util.GenerateRandomName(3),
		Config: config,

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}),
	}

	statsManager := NewStatsManager(node)
	node.StatsManager = statsManager

	nodeProfiler, err := profiler.NewProfiler()
	if err != nil {
		logrus.Fatalf("error creating profiler: %v", err)
	}
	node.Profiler = nodeProfiler

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Host: "localhost",
		Port: "2379",
	})
	if err != nil {
		logrus.Fatalf("error connecting to etcd: %v", err)
	}
	node.EtcdClient = etcdClient

	freePortForGrpc, err := util.GetFreePort()
	if err != nil {
		logrus.Fatalf("could not resolve free random port for grpc: %v", err)
	}
	grpcServer, err := NewGrpcServer(GrpcServerConfig{
		Host: "localhost",
		Port: fmt.Sprintf("%v", freePortForGrpc),
	}, node)
	if err != nil {
		logrus.Errorf("error creating node grpc server: %v", err)
	}
	node.GrpcServer = grpcServer

	return node, nil
}

func (n *Node) CollectMetrics() (*transmission_pb.NodeMetrics, error) {

	v, err := n.Profiler.GetMem()
	if err != nil {
		logrus.Errorf("error getting memory stats: %v", err)
	}

	usage := n.Profiler.GetPathDiskUsage("/")

	metrics := &transmission_pb.NodeMetrics{
		Capacity:       usage.Total,
		Remaining:      usage.Free,
		UsedPercent:    float32(usage.UsedPercent),
		LastUpdate:     time.Now().Unix(),
		MemUsedPercent: float32(v.UsedPercent),
	}

	return metrics, nil
}

func (n *Node) Start() {

	go n.GrpcServer.Run()

	go n.handleSignals()

	err := n.Register()
	if err != nil {
		logrus.Fatalf("can't register node in etcd: %v", err)
	}

	logrus.Infof("Node %v is ready", n.Name)

	for {
		select {
		case <-n.NotifyShutdown():

			//err := n.Unregister()
			//if err != nil {
			//	logrus.Errorf("error unregistering node: %v", err)
			//}
			//return

		}
	}

}

func (n *Node) Register() error {
	info := &transmission_pb.NodeInfo{
		Name: n.Name,
		Address: &transmission_pb.NodeInfo_Address{
			Host: n.GrpcServer.Config.Host,
			Port: n.GrpcServer.Config.Port,
		},
		Metrics: &transmission_pb.NodeMetrics{
			Capacity:   10000,
			Remaining:  5000,
			LastUpdate: time.Now().Add(-time.Hour * 24 * 30).Unix(),
		},
		Location: "/dir/1",
	}
	return n.EtcdClient.RegisterNode(n.Name, info)
}

func (n *Node) Unregister() error {
	return n.EtcdClient.UnregisterNode(n.Name)
}

func (n *Node) NotifyShutdown() <-chan struct{} {
	return n.shutdownCh
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

			err := n.Unregister()
			if err != nil {
				logrus.Errorf("error unregistering node: %v", err)
			}

			n.shutdownCh <- struct{}{}

			if pidFile != "" {
				err := os.Remove(pidFile)
				if err != nil {
					logrus.Errorf("error removing pid file: %v", err)
				}
			}
			os.Exit(0)
		}
	}

}
