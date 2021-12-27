package registry

import (
	"context"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	"github.com/parasource/rhosus/rhosus/registry/recovery"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type NodesMap struct {
	registry *Registry

	mu              sync.RWMutex
	nodes           map[string]*node_pb.NodeInfo
	nodesProbeTries map[string]int
	onNodeDown      func(ctx context.Context)

	noAliveNodesCh chan struct{}

	RecoveryManager *recovery.Manager
}

func NewNodesMap(registry *Registry) *NodesMap {
	return &NodesMap{
		registry: registry,

		noAliveNodesCh: make(chan struct{}, 1),
	}
}

func (m *NodesMap) SetOnNodeDown(fun func(c context.Context)) {
	m.onNodeDown = fun
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

func (m *NodesMap) WatchNodes() {

	ticker := tickers.SetTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:
			nodes := make(map[string]*node_pb.NodeInfo, len(m.nodes))

			m.mu.RLock()
			for uid, info := range m.nodes {
				nodes[uid] = info
			}
			m.mu.RUnlock()

			aliveNodes := 0
			wg := sync.WaitGroup{}

			for uid, node := range nodes {
				wg.Add(1)
				go func(uid string, node *node_pb.NodeInfo) {

					err := m.ProbeNode(node)
					if err != nil {
						// Node does not respond to health probes

						m.mu.RLock()
						tries, ok := m.nodesProbeTries[uid]
						m.mu.RUnlock()

						if (ok && tries >= 6) || !ok {

							if m.onNodeDown != nil {
								m.onNodeDown(context.Background())
							}

							err := m.StartRecoveryProcess(node)
							if err != nil {
								logrus.Errorf("error starting node recoverying proccess: %v", err)
								return
							}
						}

						m.mu.Lock()
						if ok {
							tries++
						} else {
							m.nodesProbeTries[uid] = 0
						}
						m.mu.Unlock()

					}

					m.mu.Lock()
					aliveNodes++
					m.mu.Unlock()

				}(uid, node)
			}

			wg.Wait()

			if aliveNodes <= 1 {
				m.noAliveNodesCh <- struct{}{}
			}

		}
	}

}

func (m *NodesMap) ProbeNode(node *node_pb.NodeInfo) error {

	// todo

	return nil
}

func (m *NodesMap) StartRecoveryProcess(node *node_pb.NodeInfo) error {

	// todo

	return nil
}

func (m *NodesMap) NotifyNoAliveNodes() <-chan struct{} {
	return m.noAliveNodesCh
}
