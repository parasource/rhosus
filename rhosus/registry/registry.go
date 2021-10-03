package registry

import (
	"github.com/gomodule/redigo/redis"
	rlog "github.com/parasource/rhosus/rhosus/logging"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type RegistryConfig struct {
	RpcAddress  string
	HttpAddress string
}

type Registry struct {
	mu     sync.RWMutex
	config RegistryConfig

	Log *rlog.LogHandler

	Storage RegistryStorage
	Broker  RegistryBroker

	RegistriesMap *RegistriesMap
	NodesMap      *NodesMap

	shutdownCh chan struct{}

	lastUpdate uint64
}

func NewRegistry(config RegistryConfig) (*Registry, error) {

	r := &Registry{
		config: config,
		Log:    rlog.NewLogHandler(),

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
	shardsPool.SetMessagesHandler(func(message redis.Message) {
		switch message.Channel {
		case rhosus_redis.RegistryInfoChannel:
			logrus.Infof("node info message received")
		case rhosus_redis.PingChannel:
		default:
			logrus.Infof("message from unknown channel %v", message.Channel)
		}
	})
	shardsPool.Run()

	storage, err := NewRedisRegistryStorage(shardsPool)
	if err != nil {
		return nil, err
	}

	broker, err := NewRegistryRedisBroker(shardsPool)
	if err != nil {
		return nil, err
	}

	rMap := NewRegistriesMap(r)
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

	r.sendPing()

	return err
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
		RpcAddress:  r.config.RpcAddress,
		HttpAddress: r.config.HttpAddress,
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
