package registry

import (
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
	file_server "github.com/parasource/rhosus/rhosus/server"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type RegistryConfig struct {
	HttpHost string
	HttpPort string

	ServerConfig file_server.ServerConfig
}

type Registry struct {
	Name   string
	mu     sync.RWMutex
	Config RegistryConfig

	Storage  RegistryStorage
	IsLeader bool

	RegistriesMap  *RegistriesMap
	NodesMap       *NodesMap
	FileServer     *file_server.Server
	StatsCollector *StatsCollector
	etcdClient     *rhosus_etcd.EtcdClient

	readyCh chan struct{}
	readyWg sync.WaitGroup

	shutdownCh chan struct{}

	lastUpdate uint64
}

func NewRegistry(config RegistryConfig) (*Registry, error) {

	r := &Registry{
		Name:    util.GenerateRandomName(2),
		Config:  config,
		readyWg: sync.WaitGroup{},

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}),
	}

	storage, err := NewEtcdStorage(EtcdStorageConfig{}, r)
	if err != nil {
		logrus.Fatalf("error creating etcd storage: %v", err)
	}
	r.Storage = storage

	rMap := NewRegistriesMap()
	r.RegistriesMap = rMap

	nMap := NewNodesMap(r)
	r.NodesMap = nMap

	statsCollector := NewStatsCollector(r, 5)
	r.StatsCollector = statsCollector

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Host: "localhost",
		Port: "2379",
	})
	if err != nil {
		logrus.Fatalf("error connecting to etcd: %v", err)
	}
	r.etcdClient = etcdClient

	fileServer, err := file_server.NewServer(file_server.ServerConfig{
		Host:      r.Config.HttpHost,
		Port:      r.Config.HttpPort,
		MaxSizeMb: 500,
	})
	if err != nil {
		logrus.Fatalf("error starting file server: %v", err)
	}
	r.FileServer = fileServer

	return r, nil
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) Start() {

	go r.RegistriesMap.RunCleaning()

	go r.FileServer.RunHTTP()

	go r.NodesMap.WatchNodes()
	go r.StatsCollector.Run()

	go r.handleSignals()
	go r.RunServiceDiscovery()

	// Blocks goroutine until ready signal received
	close(r.readyCh)

	err := r.register()
	if err != nil {
		logrus.Errorf("error registering in etcd: %v", err)
	}

	logrus.Infof("Registry %v is ready", r.Name)

	for {
		select {
		case <-r.NotifyShutdown():

			r.FileServer.SendShutdownSignal()

			err := r.unregister()
			if err != nil {
				logrus.Errorf("error unregistering: %v", err)
			}

			return
		}
	}

}

func (r *Registry) register() error {
	info := &registry_pb.RegistryInfo{
		Name: r.Name,
		HttpAddress: &registry_pb.RegistryInfo_Address{
			Host: r.Config.HttpHost,
			Port: r.Config.HttpPort,
		},
	}

	return r.etcdClient.RegisterRegistry(r.Name, info)
}

func (r *Registry) unregister() error {
	return r.etcdClient.UnregisterRegistry(r.Name)
}

func (r *Registry) NotifyShutdown() <-chan struct{} {
	return r.shutdownCh
}

func (r *Registry) NotifyReady() <-chan struct{} {
	return r.readyCh
}

func (r *Registry) handleSignals() {
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

			r.shutdownCh <- struct{}{}

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

////////////////////////////
// Nodes and registries management methods

func (r *Registry) RunServiceDiscovery() {

	// Load existing nodes

	nodes, err := r.etcdClient.GetExistingNodes()
	if err != nil {
		logrus.Fatalf("error getting existing nodes: %v", err)
	}
	for path, bytes := range nodes {

		name := rhosus_etcd.ParseNodeName(path)

		var info transmission_pb.NodeInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling node info: %v", err)
			continue
		}

		err = r.NodesMap.AddNode(name, &info)
		if err != nil {
			logrus.Errorf("error adding node to map: %v", err)
			continue
		}

		logrus.Infof("node %v added from existing map", info.Name)
	}

	registries, err := r.etcdClient.GetExistingRegistries()
	if err != nil {
		logrus.Fatalf("error getting existing registries: %v", err)
	}
	for path, bytes := range registries {

		name := rhosus_etcd.ParseRegistryName(path)

		if name == r.Name {
			continue
		}

		var info registry_pb.RegistryInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling node info: %v", err)
			continue
		}

		err = r.RegistriesMap.Add(name, &info)
		if err != nil {
			logrus.Errorf("error adding registry to map: %v", err)
			continue
		}

		logrus.Infof("registry %v added from existing map", info.Name)
	}

	// Waiting for new nodes to connect

	ticker := tickers.SetTicker(time.Second * 3)

	for {
		select {
		case <-ticker.C:

			err := r.etcdClient.Ping()
			if err != nil {
				// todo
			}

		case res := <-r.etcdClient.WatchForNodesUpdates():
			// Here we handle nodes updates

			for _, event := range res.Events {

				switch event.Type {

				// Node added or updated
				case clientv3.EventTypePut:

					name := rhosus_etcd.ParseNodeName(string(event.Kv.Key))
					data := string(event.Kv.Value)

					bytes, err := util.Base64Decode(data)
					if err != nil {
						logrus.Errorf("error decoding")
					}

					var info transmission_pb.NodeInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						logrus.Errorf("error unmarshaling node info: %v", err)
					}

					if r.NodesMap.NodeExists(name) {
						// If node already exists in a node map => updating node info

						r.NodesMap.UpdateNodeInfo(name, &info)
					} else {
						err := r.NodesMap.AddNode(name, &info)
						if err != nil {
							logrus.Errorf("error adding node: %v", err)
							continue
						}

						logrus.Infof("node %v added", info.Name)
					}

				// Node shut down or errored
				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseNodeName(string(event.Kv.Key))

					if r.NodesMap.NodeExists(name) {
						r.NodesMap.RemoveNode(name)

						logrus.Infof("node %v shut down", name)
					} else {
						logrus.Warnf("undefined node deletion signal")
					}

				}
			}

		case res := <-r.etcdClient.WatchForRegistriesUpdates():
			// Here we handle registries updates

			for _, event := range res.Events {

				switch event.Type {

				case clientv3.EventTypePut:

					name := string(event.Kv.Key)
					data := string(event.Kv.Value)

					if name == r.Name {
						continue
					}

					bytes, err := util.Base64Decode(data)
					if err != nil {
						logrus.Errorf("error decoding")
					}

					var info registry_pb.RegistryInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						logrus.Errorf("error unmarshaling registry info: %v", err)
					}

					if r.RegistriesMap.RegistryExists(name) {

					} else {
						err := r.RegistriesMap.Add(name, &info)
						if err != nil {
							logrus.Errorf("error adding new registry: %v", err)
						}
					}

				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseRegistryName(string(event.Kv.Key))

					if name == r.Name {
						continue
					}

					if r.RegistriesMap.RegistryExists(name) {
						err := r.RegistriesMap.Remove(name)
						if err != nil {
							logrus.Errorf("error removing registry: %v", err)
						}
					} else {
						logrus.Warn("undefined registry deletion signal")
					}

				}
			}

		case <-r.NotifyShutdown():

			return
		}
	}
}
