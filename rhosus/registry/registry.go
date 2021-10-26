package registry

import (
	"context"
	"github.com/gomodule/redigo/redis"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
	file_server "github.com/parasource/rhosus/rhosus/server"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
	"sync"
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

	Storage RegistryStorage
	Broker  RegistryBroker

	RegistriesMap *RegistriesMap
	NodesMap      *NodesMap

	shutdownCh chan struct{}

	lastUpdate uint64
}

func NewRegistry(config RegistryConfig) (*Registry, error) {

	uid, err := uuid.NewV4()
	if err != nil {
		logrus.Fatalf("could not generate uid for registry instance: %v", err)
	}
	r := &Registry{
		Uid:    uid.String(),
		Config: config,

		shutdownCh: make(chan struct{}, 1),
	}

	shardsPool, err := rhosus_redis.NewRedisShardPool([]rhosus_redis.RedisShardConfig{
		{
			Host: "127.0.0.1",
			Port: 6379,
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

	return r, nil
}

func (r *Registry) NotifyShutdown() <-chan struct{} {
	return r.shutdownCh
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

func (r *Registry) Run() error {
	var err error

	// Here will be grpc server. Probably.

	// running cleaning process. It watches if other registries are still alive
	go r.RegistriesMap.RunCleaning()

	go r.runHttpFileServer()

	r.sendPing()

	return err
}

func (r *Registry) runHttpFileServer() {

	server, err := file_server.NewServer(file_server.ServerConfig{
		Host:      strings.Split(r.Config.HttpAddress, ":")[0],
		Port:      strings.Split(r.Config.HttpAddress, ":")[1],
		MaxSizeMb: 5000,
	})
	server.SetRegistryAddFunc(r.RegisterFile)
	server.SetRegistryDeleteFunc(r.DeleteFile)

	if err != nil {
		return
	}

	httpServer := &http.Server{
		Addr:              net.JoinHostPort(server.Config.Host, server.Config.Port),
		Handler:           http.HandlerFunc(server.Handle),
		TLSConfig:         nil,
		ReadTimeout:       0,
		ReadHeaderTimeout: 0,
		WriteTimeout:      0,
		IdleTimeout:       0,
		MaxHeaderBytes:    0,
		TLSNextProto:      nil,
		ConnState:         nil,
		ErrorLog:          nil,
		BaseContext:       nil,
		ConnContext:       nil,
	}

	if err := httpServer.ListenAndServe(); err != nil {
		logrus.Fatalf("error starting file http server: %v", err)
	}

	logrus.Infof("http file server is up and running")

	for {
		select {
		case <-r.NotifyShutdown():
			httpServer.Shutdown(context.Background())
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

	err = r.Broker.PublishCommand(pubReq)
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

////////////////////////////
// Nodes management methods

func (r *Registry) AddNode(uid string, info *node_pb.NodeInfo) {
	r.NodesMap.AddNode(uid, info)
}

func (r *Registry) RemoveNode(uid string) {
	r.NodesMap.RemoveNode(uid)
}

func (r *Registry) Shutdown() error {

	r.shutdownCh <- struct{}{}

	return nil
}
