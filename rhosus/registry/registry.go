package registry

import (
	rhosus_node "github.com/parasource/rhosus/rhosus/node"
	"sync"
)

type RegistryConfig struct {
	address string
}

type Registry struct {
	config RegistryConfig

	nodesMu sync.Mutex
	Storage RegistryStorage
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
	r.Storage = storage

	return r, nil
}

func (r *Registry) Run() {
	err := r.Storage.Run()
	if err != nil {

	}

}

func (r *Registry) NotifyShutdown() <-chan struct{} {
	return r.closeCh
}

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
