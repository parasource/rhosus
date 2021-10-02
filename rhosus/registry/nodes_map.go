package registry

import (
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	"sync"
)

var (
	addNodeSource = `redis.call("HSET", KEYS[1], ARGV[1], ARGV[2])`

	removeNodeSource = `redis.call("HDEL", KEYS[1], ARGV[1])`

	getNodesSource = `return redis.call("HDELALL", KEYS[1])`
)

type NodesMap struct {
	registry *Registry

	mu    sync.RWMutex
	nodes map[string]*node_pb.NodeInfo
}

func NewNodesMap(registry *Registry) *NodesMap {
	return &NodesMap{
		registry: registry,
	}
}

func (m *NodesMap) AddNode(uid string, info *node_pb.NodeInfo) {
	m.mu.Lock()
	if _, ok := m.nodes[uid]; !ok {
		m.nodes[uid] = info
	}
	m.mu.Unlock()

	m.registry.Storage.GetNodes()
}

func (m *NodesMap) RemoveNode(uid string) {
	m.mu.Lock()
	if _, ok := m.nodes[uid]; ok {
		delete(m.nodes, uid)
	}
	m.mu.Unlock()
}

func (m *NodesMap) List() map[string]*node_pb.NodeInfo {
	return nil
}
