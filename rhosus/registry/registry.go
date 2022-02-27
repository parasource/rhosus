package registry

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/backend"
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/registry/cluster"
	file_server "github.com/parasource/rhosus/rhosus/server"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	etcdPingInterval = 1
	uuidFilePath     = "uuid"
)

type Config struct {
	ClusterHost     string `json:"cluster_host"`
	ClusterPort     string `json:"cluster_port"`
	ClusterUsername string `json:"cluster_username"`
	ClusterPassword string `json:"cluster_password"`

	ServerConfig file_server.ServerConfig
}

type Registry struct {
	Id     string
	Name   string
	mu     sync.RWMutex
	Config Config

	IsLeader bool

	NodesManager   *NodesMap
	FileServer     *file_server.Server
	Backend        *backend.Storage
	MemoryStorage  *MemoryStorage
	StatsCollector *StatsCollector

	// Cluster is used to control over other registries
	Cluster *cluster.Cluster

	etcdClient *rhosus_etcd.EtcdClient

	readyC  chan struct{}
	readyWg sync.WaitGroup

	shutdownC chan struct{}
}

func NewRegistry(config Config) (*Registry, error) {

	id := getId(false)

	r := &Registry{
		Id:      id,
		Name:    util.GenerateRandomName(2),
		Config:  config,
		readyWg: sync.WaitGroup{},

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

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

	s, err := backend.NewStorage(backend.Config{
		WriteTimeoutS: 1,
	})
	if err != nil {
		logrus.Fatalf("error creating storage: %v", err)
	}
	r.Backend = s

	memStorage, err := NewMemoryStorage(r)
	if err != nil {
		logrus.Fatalf("error creating memory storage: %v", err)
	}
	r.MemoryStorage = memStorage

	// Here we load all the existing nodes and registries from etcd
	// Error occurs only in non-usual conditions, so we kill process
	regs, err := r.getExistingRegistries()
	if err != nil {
		logrus.Fatalf("error getting existing registries from etcd: %v", err)
	}

	nodes, err := r.getExistingNodes()
	if err != nil {
		logrus.Fatalf("error getting existing nodes from etcd: %v", err)
	}

	port, err := util.GetFreePort()
	if err != nil {
		logrus.Fatalf("couldn't get free port: %v", err)
	}
	info := &control_pb.RegistryInfo{
		Id:   r.Id,
		Name: r.Name,
		Address: &control_pb.RegistryInfo_Address{
			Host:     "localhost",
			Port:     fmt.Sprintf("%v", port),
			Username: "",
			Password: "",
		},
	}

	// Setting up registries cluster from existing peers
	c := cluster.NewCluster(cluster.Config{
		ID:           r.Id,
		RegistryInfo: info,
	}, regs)
	r.Cluster = c

	// Setting up nodes map from existing nodes
	nMap, err := NewNodesMap(r, nodes)
	r.NodesManager = nMap

	// Registering itself in etcd cluster
	err = r.registerItself(info)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	fileServer, err := file_server.NewServer(file_server.ServerConfig{
		Host:      r.Config.ServerConfig.Host,
		Port:      r.Config.ServerConfig.Port,
		MaxSizeMb: 500,
	})
	if err != nil {
		logrus.Fatalf("error starting file server: %v", err)
	}
	r.FileServer = fileServer

	return r, nil
}

func getId(persistent bool) string {
	var id string

	if !persistent {
		v4id, _ := uuid.NewV4()
		return v4id.String()
	}

	// since we are just testing, we don't need that yet
	if util.FileExists(uuidFilePath) {
		file, err := os.OpenFile(uuidFilePath, os.O_RDONLY, 0666)
		defer file.Close()

		if err != nil {
			logrus.Errorf("error opening node uuid file: %v", err)
		}
		data, err := io.ReadAll(file)
		if err != nil {

		}

		id = string(data)
	} else {
		v4uid, _ := uuid.NewV4()
		id = v4uid.String()

		file, err := os.Create(uuidFilePath)
		defer file.Close()

		if err != nil {

		}
		file.Write([]byte(id))
	}

	return id
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) Start() {

	//var err error

	go r.FileServer.RunHTTP()

	go r.NodesManager.WatchNodes()
	go r.StatsCollector.Run()

	go r.RunServiceDiscovery()
	go r.MemoryStorage.Start()

	go r.handleSignals()

	go func() {
		<-time.After(time.Second * 2)

		uid, _ := uuid.NewV4()

		file := &control_pb.FileInfo{
			Id:    uid.String(),
			Type:  control_pb.FileInfo_FILE,
			Path:  "test/index.html",
			Size_: 128000,
			Permission: &control_pb.FileInfo_FsPermission{
				Perm: 0,
			},
			Owner: "egorovchinnikov",
			Group: "owner",
		}

		var data []byte
		for j := 0; j < 4*1024*1000; j++ {
			data = append(data, byte('b'))
		}
		var blocks []*fs_pb.Block
		for i := 0; i < 256; i++ {
			uid, _ := uuid.NewV4()

			blocks = append(blocks, &fs_pb.Block{
				Id:     uid.String(),
				FileId: "testfile123",
				Size_:  16,
				Data:   data,
			})
		}

		_ = r.registerFileWithBlocks(file, blocks)

	}()

	r.readyC <- struct{}{}

	logrus.Infof("Registry %v:%v is ready", r.Name, r.Id)

	if <-r.NotifyShutdown(); true {

		logrus.Infof("shutting down registry")
		pidFile := viper.GetString("pid_file")
		err := r.unregisterItself()
		if err != nil {
			logrus.Errorf("error unregistering: %v", err)
		}

		r.FileServer.Shutdown()
		r.Backend.Shutdown()
		r.Cluster.Shutdown()

		if pidFile != "" {
			err := os.Remove(pidFile)
			if err != nil {
				logrus.Errorf("error removing pid file: %v", err)
			}
		}
		os.Exit(0)
		return
	}

}

func (r *Registry) registerItself(info *control_pb.RegistryInfo) error {
	return r.etcdClient.RegisterRegistry(info.Id, info)
}

func (r *Registry) unregisterItself() error {
	return r.etcdClient.UnregisterRegistry(r.Id)
}

func (r *Registry) NotifyShutdown() <-chan struct{} {
	return r.shutdownC
}

func (r *Registry) NotifyReady() <-chan struct{} {
	return r.readyC
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

			pidFile := viper.GetString("pid_file")
			shutdownTimeout := time.Duration(viper.GetInt("shutdown_timeout")) * time.Second

			close(r.shutdownC)

			go time.AfterFunc(shutdownTimeout, func() {
				if pidFile != "" {
					os.Remove(pidFile)
				}
				os.Exit(1)
			})
		}
	}

}

func (r *Registry) getExistingRegistries() (map[string]*control_pb.RegistryInfo, error) {

	registries, err := r.etcdClient.GetExistingRegistries()
	if err != nil {
		logrus.Fatalf("error getting existing registries: %v", err)
	}

	result := make(map[string]*control_pb.RegistryInfo)

	for _, bytes := range registries {

		var info control_pb.RegistryInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling node info: %v", err)
			continue
		}

		result[info.Id] = &info

		logrus.Infof("registry %v added from existing map", info.Name)
	}

	return result, nil
}

func (r *Registry) getExistingNodes() (map[string]*transport_pb.NodeInfo, error) {

	nodes, err := r.etcdClient.GetExistingNodes()
	if err != nil {
		logrus.Fatalf("error getting existing registries: %v", err)
	}

	result := make(map[string]*transport_pb.NodeInfo)

	for _, bytes := range nodes {

		var info transport_pb.NodeInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling node info: %v", err)
			continue
		}

		result[info.Id] = &info
	}

	return result, nil
}

// <------------------------------------->
// Nodes and registries management methods
// <------------------------------------->

func (r *Registry) RunServiceDiscovery() {

	// Waiting for new nodes to connect

	ticker := tickers.SetTicker(time.Second * time.Duration(etcdPingInterval))
	defer ticker.Stop()

	for {

		select {
		case <-ticker.C:

			err := r.etcdClient.Ping()
			if err != nil {
				logrus.Errorf("error pinging etcd: %v", err)
			}

		case res := <-r.etcdClient.WatchForNodesUpdates():
			// Here we handle nodes updates

			for _, event := range res.Events {

				switch event.Type {

				// Node added or updated
				case clientv3.EventTypePut:

					data := string(event.Kv.Value)

					bytes, err := util.Base64Decode(data)
					if err != nil {
						logrus.Errorf("error decoding")
					}

					var info transport_pb.NodeInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						logrus.Errorf("error unmarshaling node info: %v", err)
					}

					if r.NodesManager.NodeExists(info.Id) {
						// If node already exists in a node map => updating node info

						r.NodesManager.UpdateNodeInfo(info.Id, &info)
					} else {
						err := r.NodesManager.AddNode(info.Id, &info)
						if err != nil {
							logrus.Errorf("error adding node: %v", err)
							continue
						}

						logrus.Infof("node %v added", info.Name)
					}

				// Node shut down or errored
				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseNodeName(string(event.Kv.Key))

					if r.NodesManager.NodeExists(name) {
						r.NodesManager.RemoveNode(name)

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

					var info control_pb.RegistryInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						logrus.Errorf("error unmarshaling registry info: %v", err)
					}

					err = r.Cluster.DiscoverOrUpdate(info.Id, &info)
					if err != nil {
						logrus.Errorf("error discovering registry: %v", err)
					}
					//if r.RegistriesMap.RegistryExists(name) {
					//
					//} else {
					//	err := r.RegistriesMap.Add(name, &info)
					//	if err != nil {
					//		logrus.Errorf("error adding new registry: %v", err)
					//	}
					//}

				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseRegistryName(string(event.Kv.Key))

					if name == r.Name {
						continue
					}

					//if r.RegistriesMap.RegistryExists(name) {
					//	err := r.RegistriesMap.Remove(name)
					//	if err != nil {
					//		logrus.Errorf("error removing registry: %v", err)
					//	}
					//} else {
					//	logrus.Warn("undefined registry deletion signal")
					//}

				}
			}
		}
	}
}
