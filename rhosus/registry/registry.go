package registry

import (
	"context"
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
	file_server "github.com/parasource/rhosus/rhosus/server"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type RegistryConfig struct {
	HttpHost string
	HttpPort string
	GrpcHost string
	GrpcPort string

	ServerConfig file_server.ServerConfig
}

type Registry struct {
	Uid    string
	mu     sync.RWMutex
	Config RegistryConfig

	Storage  RegistryStorage
	Broker   RegistryBroker
	IsLeader bool

	RegistriesMap *RegistriesMap
	NodesMap      *NodesMap
	FileServer    *file_server.Server
	NodeServer    *NodeServer

	readyCh chan struct{}
	readyWg sync.WaitGroup

	shutdownCh chan struct{}

	lastUpdate uint64
}

func NewRegistry(config RegistryConfig) (*Registry, error) {

	uid, err := uuid.NewV4()
	if err != nil {
		logrus.Fatalf("could not generate uid for registry instance: %v", err)
	}
	r := &Registry{
		Uid:     uid.String(),
		Config:  config,
		readyWg: sync.WaitGroup{},

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}),
	}

	//storage, err := NewRedisRegistryStorage(shardsPool)
	//if err != nil {
	//	return nil, err
	//}

	storage, err := NewEtcdStorage(EtcdStorageConfig{}, r)
	if err != nil {
		logrus.Fatalf("error creating etcd storage: %v", err)
	}
	r.Storage = storage

	rMap := NewRegistriesMap()
	r.RegistriesMap = rMap

	nMap := NewNodesMap(r)
	r.NodesMap = nMap

	fileServer, err := file_server.NewServer(file_server.ServerConfig{
		Host:      r.Config.HttpHost,
		Port:      r.Config.HttpPort,
		MaxSizeMb: 500,
	})
	if err != nil {
		logrus.Fatalf("error starting file server: %v", err)
	}
	r.FileServer = fileServer

	// todo: move to config
	nodeServer, err := NewNodeServer(NodeServerConfig{
		Host: "localhost",
		Port: "6435",
	}, r)
	if err != nil {
		logrus.Fatalf("error creating grpc node server: %v", err)
	}
	r.NodeServer = nodeServer

	return r, nil
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) AddRegistry(uid string, info *registry_pb.RegistryInfo) {
	r.RegistriesMap.Add(uid, info)
}

func (r *Registry) RemoveRegistry(uid string) {
	r.RegistriesMap.Remove(uid)
}

func (r *Registry) Start() {

	// Here will be grpc server. Probably.

	// running cleaning process. It watches if other registries are still alive
	go r.RegistriesMap.RunCleaning()

	// http file server
	go r.FileServer.RunHTTP()
	r.readyWg.Add(1)
	go func() {
		<-r.FileServer.NotifyReady()

		r.readyWg.Done()
	}()

	go r.NodeServer.Run()
	r.readyWg.Add(1)
	go func() {
		<-r.NodeServer.NotifyReady()

		r.readyWg.Done()
	}()

	go r.NodesMap.WatchNodes()

	// handle signals for grace shutdown
	go r.handleSignals()

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Host: "localhost",
		Port: "2379",
	})
	if err != nil {
		logrus.Fatalf("error connecting to etcd: %v", err)
	}

	go r.RunServiceDiscovery(etcdClient)

	// Blocks goroutine until ready signal received
	r.readyWg.Wait()
	logrus.Infof("Registry %v is ready", r.Uid)
	close(r.readyCh)

	for {
		select {
		case <-r.NotifyShutdown():

			r.FileServer.SendShutdownSignal()

			return
		}
	}

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
			time.Sleep(time.Second)
			os.Exit(0)
		}
	}

}

////////////////////////////
// Nodes management methods

func (r *Registry) RunServiceDiscovery(etcdClient *rhosus_etcd.EtcdClient) {

	ticker := tickers.SetTicker(time.Second * 3)
	counter := 0

	for {
		select {
		case <-ticker.C:

			value := strconv.Itoa(counter)
			_, err := etcdClient.Put(context.Background(), "counter", value)
			if err != nil {
				logrus.Errorf("error putting value to etcd: %v", err)
			}
			counter++

		case res := <-etcdClient.WatchForNodesUpdates():
			// Here we handle nodes updates

			for _, event := range res.Events {
				uid := string(event.Kv.Key)
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

				if r.NodesMap.NodeExists(uid) {

				} else {
					r.NodesMap.AddNode(uid, &info)

					logrus.Infof("node %v added", info.Uid)
				}
			}

		case res := <-etcdClient.WatchForRegistriesUpdates():
			// Here we handle registries updates

			for _, event := range res.Events {
				uid := string(event.Kv.Key)
				data := string(event.Kv.Value)

				bytes, err := util.Base64Decode(data)
				if err != nil {
					logrus.Errorf("error decoding")
				}

				var info registry_pb.RegistryInfo
				err = info.Unmarshal(bytes)
				if err != nil {
					logrus.Errorf("error unmarshaling registry info: %v", err)
				}

				if r.RegistriesMap.RegistryExists(uid) {

				} else {
					r.RegistriesMap.Add(uid, &info)
				}
			}

		case <-r.NotifyShutdown():

			return
		}
	}
}
