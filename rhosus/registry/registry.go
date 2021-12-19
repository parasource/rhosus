package registry

import (
	"github.com/gomodule/redigo/redis"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
	file_server "github.com/parasource/rhosus/rhosus/server"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type RegistryConfig struct {
	RpcAddress  string
	HttpAddress string

	fileHttpServer *file_server.Server
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

	readyCh chan struct{}
	readyWg sync.WaitGroup

	shutdownCh chan struct{}

	lastUpdate uint64
}

func NewRegistry(config RegistryConfig) (*Registry, error) {

	v := viper.GetViper()

	uid, err := uuid.NewV4()
	if err != nil {
		logrus.Fatalf("could not generate uid for registry instance: %v", err)
	}
	r := &Registry{
		Uid:     uid.String(),
		Config:  config,
		readyWg: sync.WaitGroup{},

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}, 1),
	}

	shardsPool, err := rhosus_redis.NewRedisShardPool([]rhosus_redis.RedisShardConfig{
		{
			Host: v.GetString("redis_host"),
			Port: v.GetInt("redis_port"),
		},
	})
	if err != nil {
		return nil, err
	}

	// Here we set a message handler for incoming messages.
	// Basically this should be done inside shard pool,
	// but since we have no other drivers yet - it's fine
	shardsPool.SetMessagesHandler(r.HandleBrokerMessages)
	shardsPool.Run()
	r.readyWg.Add(1)
	go func() {
		for {
			select {
			case <-shardsPool.NotifyReady():
				r.readyWg.Done()
			}
		}
	}()

	storage, err := NewRedisRegistryStorage(shardsPool)
	if err != nil {
		return nil, err
	}

	broker, err := NewRegistryRedisBroker(shardsPool)
	if err != nil {
		return nil, err
	}

	rMap := NewRegistriesMap()
	r.RegistriesMap = rMap

	nMap := NewNodesMap(r)
	r.NodesMap = nMap

	r.Storage = storage
	r.Broker = broker

	fileServer, err := file_server.NewServer(file_server.ServerConfig{
		Host:      "localhost",
		Port:      "8011",
		MaxSizeMb: 500,
	})
	if err != nil {
		logrus.Fatalf("error starting file server: %v", err)
	}

	r.FileServer = fileServer

	return r, nil
}

/////////////////////////////////////
// Handlers
//
// Here I define some handlers, that I
// can use only this way. It is because
// of impossibility of importing
// packages one into another

// HandleBrokerMessages handles broker messages
func (r *Registry) HandleBrokerMessages(message redis.Message) {
	switch message.Channel {
	case rhosus_redis.RegistryInfoChannel:
		r.handleRegistryInfo(message.Data)
	case rhosus_redis.PingChannel:
	default:
		logrus.Infof("message from unknown channel %v", message.Channel)
	}
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) AddRegistry(uid string, info *registry_pb.RegistryInfo) {
	r.RegistriesMap.Add(uid, info)
}

func (r *Registry) RemoveRegistry(uid string) {
	r.RegistriesMap.Remove(uid)
}

func (r *Registry) Run() {

	// Here will be grpc server. Probably.

	// running cleaning process. It watches if other registries are still alive
	go r.RegistriesMap.RunCleaning()

	// http file server
	go r.FileServer.RunHTTP()
	r.readyWg.Add(1)
	go func() {
		for {
			select {
			case <-r.FileServer.NotifyReady():
				r.readyWg.Done()
			}
		}
	}()

	// ping process to show that registry is still alive
	go r.sendPing()

	// handle signals for grace shutdown
	go r.handleSignals()
	go r.listenReady()

	for {
		select {
		case <-r.NotifyReady():

			logrus.Infof("Registry %v is ready", r.Uid)

		case <-r.NotifyShutdown():

			r.FileServer.SendShutdownSignal()

			return
		}
	}

}

func (r *Registry) sendPing() {
	ticker := time.NewTicker(time.Second * 5)

	for {
		select {
		case <-r.NotifyShutdown():
			return
		case <-ticker.C:
			r.pubRegistryInfo()
		}
	}
}

func (r *Registry) pubRegistryInfo() {
	r.mu.RLock()

	info := registry_pb.RegistryInfo{
		Uid:         r.Uid,
		RpcAddress:  r.Config.RpcAddress,
		HttpAddress: r.Config.HttpAddress,
	}

	r.mu.RUnlock()

	bytes, err := info.Marshal()
	if err != nil {
		logrus.Fatalf("error marshaling registry info: %v", err)
		return
	}

	command := &registry_pb.Command{
		Type: registry_pb.Command_PING,
		Data: bytes,
	}

	bytesCmd, err := command.Marshal()
	if err != nil {
		logrus.Fatalf("error marshaling registry command: %v", err)
		return
	}

	pubReq := rhosus_redis.PubRequest{
		Channel: rhosus_redis.RegistryInfoChannel,
		Data:    bytesCmd,
		Err:     make(chan error, 1),
	}

	err = r.Broker.Publish(pubReq)
	if err != nil {
		logrus.Fatalf("error publishing registry command: %v", err)
	}

}

func (r *Registry) handleRegistryInfo(data []byte) {
	var cmd registry_pb.Command
	err := cmd.Unmarshal(data)
	if err != nil {
		logrus.Errorf("error unmarshaling command: %v", err)
	}

	var info registry_pb.RegistryInfo
	err = info.Unmarshal(cmd.Data)
	if err != nil {
		logrus.Errorf("error unmarshaling info: %v", err)
	}

	r.RegistriesMap.Add(info.Uid, &info)
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

func (r *Registry) listenReady() {

	// We are waiting til every service is running
	r.readyWg.Wait()

	r.readyCh <- struct{}{}

}

////////////////////////////
// Nodes management methods

func (r *Registry) AddNode(uid string, info *node_pb.NodeInfo) {
	r.NodesMap.AddNode(uid, info)
}

func (r *Registry) RemoveNode(uid string) {
	r.NodesMap.RemoveNode(uid)
}
