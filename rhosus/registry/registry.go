package registry

import (
	rlog "github.com/parasource/rhosus/rhosus/logging"
	rhosus_node "github.com/parasource/rhosus/rhosus/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
	"sync"
	"time"
)

const (
	registryInfoChannel = "--rhosus-registry-channel"
)

type RegistryConfig struct {
	rpcAddress  string
	httpAddress string
}

type Registry struct {
	mu     sync.RWMutex
	config RegistryConfig

	Log *rlog.LogHandler

	Storage       RegistryStorage
	Broker        RegistryBroker
	RegistriesMap *RegistriesMap

	nodesMu sync.RWMutex
	nodes   map[string]*rhosus_node.Node

	closeCh chan struct{}

	lastUpdate uint64
}

func NewRegistry(config RegistryConfig) (*Registry, error) {
	r := &Registry{
		config:  config,
		closeCh: make(chan struct{}, 1),
	}
	storage, err := NewRedisRegistryStorage(r, nil)
	if err != nil {
		return nil, err
	}

	broker, err := NewRegistryRedisBroker(r, rhosus_redis.RedisConfig{})
	if err != nil {

	}
	r.Storage = storage
	r.Broker = broker

	return r, nil
}

func (r *Registry) NotifyShutdown() <-chan struct{} {
	return r.closeCh
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) AddRegistry(uid string, info *registry_pb.RegistryInfo) {
	r.RegistriesMap.Add(uid, info)
}

func (r *Registry) RemoveRegistry(uid string) {
	r.RegistriesMap.Remove(uid)
}

func (r *Registry) sendPing() {
	ticker := time.NewTicker(time.Second * 10)

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
		RpcAddress:  r.config.rpcAddress,
		HttpAddress: r.config.httpAddress,
	}

	r.mu.RUnlock()

	bytes, err := info.Marshal()
	if err != nil {

	}

	command := &registry_pb.Command{
		Type: registry_pb.Command_PING,
		Data: bytes,
	}

	bytesCmd, err := command.Marshal()
	if err != nil {

	}

	pubReq := rhosus_redis.PubRequest{
		Channel: registryInfoChannel,
		Data:    bytesCmd,
		Err:     nil,
	}

	err = r.Broker.PublishCommand(pubReq)
	if err != nil {

	}

}

////////////////////////////
// Nodes management methods

func (r *Registry) RegisterNode(key string, node *rhosus_node.Node) error {
	r.nodesMu.Lock()
	defer r.nodesMu.Unlock()

	r.nodes[key] = node

	return nil
}

func (r *Registry) RemoveNode(key string) error {
	r.nodesMu.Lock()
	defer r.nodesMu.Unlock()

	delete(r.nodes, key)

	return nil
}

func (r *Registry) Shutdown() error {

	r.closeCh <- struct{}{}

	return nil
}
