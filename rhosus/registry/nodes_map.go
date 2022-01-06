package registry

import (
	"context"
	"errors"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sync"
	"time"
)

type NodesMap struct {
	registry *Registry

	mu              sync.RWMutex
	nodes           map[string]*NodeInfo
	nodesConns      map[string]*NodeGrpcConn
	nodesProbeTries map[string]int
	onNodeDown      func(ctx context.Context)

	noAliveNodesCh chan struct{}

	RecoveryManager *Manager
}

type NodeInfo transport_pb.NodeInfo
type NodeGrpcConn grpc.ClientConn

func NewNodesMap(registry *Registry) *NodesMap {
	return &NodesMap{
		registry: registry,

		noAliveNodesCh:  make(chan struct{}, 1),
		nodes:           make(map[string]*NodeInfo),
		nodesProbeTries: make(map[string]int),
		nodesConns:      make(map[string]*NodeGrpcConn),
	}
}

func (m *NodesMap) SetOnNodeDown(fun func(c context.Context)) {
	m.onNodeDown = fun
}

func (m *NodesMap) AddNode(name string, info *transport_pb.NodeInfo) error {

	address := net.JoinHostPort(info.Address.Host, info.Address.Port)

	logrus.Infof("establishing grpc connection with a new node on: %v", address)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	// ping new node
	client := transport_pb.NewTransportServiceClient(conn)
	_, err = client.Ping(context.Background(), &transport_pb.PingRequest{})
	if err != nil {
		return err
	}

	m.mu.Lock()
	//if _, ok := m.nodes[name]; !ok {
	m.nodes[name] = (*NodeInfo)(info)
	//}
	//if _, ok := m.nodesConns[name]; !ok {
	m.nodesConns[name] = (*NodeGrpcConn)(conn)
	//}
	m.nodesProbeTries[name] = 0
	m.mu.Unlock()

	return nil
}

func (m *NodesMap) RemoveNode(name string) {
	m.mu.Lock()
	delete(m.nodes, name)
	delete(m.nodesProbeTries, name)
	if _, ok := m.nodesConns[name]; ok {
		delete(m.nodesProbeTries, name)
	}
	m.mu.Unlock()
}

func (m *NodesMap) UpdateNodeInfo(name string, info *transport_pb.NodeInfo) {
	m.mu.Lock()
	if _, ok := m.nodes[name]; ok {
		m.nodes[name] = (*NodeInfo)(info)
	}
	m.mu.Unlock()
}

func (m *NodesMap) NodeExists(name string) bool {
	m.mu.RLock()
	_, ok := m.nodes[name]
	m.mu.RUnlock()

	return ok
}

func (m *NodesMap) GetGrpcClient(name string) (transport_pb.TransportServiceClient, error) {
	m.mu.RLock()
	conn, ok := m.nodesConns[name]
	m.mu.RUnlock()

	if !ok {
		return nil, errors.New("undefined name. node does not persist in node map")
	}

	client := transport_pb.NewTransportServiceClient((*grpc.ClientConn)(conn))
	return client, nil
}

func (m *NodesMap) WatchNodes() {

	ticker := tickers.SetTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:
			nodes := make(map[string]*transport_pb.NodeInfo, len(m.nodes))

			m.mu.RLock()
			for uid, info := range m.nodes {
				nodes[uid] = (*transport_pb.NodeInfo)(info)
			}
			m.mu.RUnlock()

			aliveNodes := 0
			wg := sync.WaitGroup{}

			for uid, node := range nodes {
				wg.Add(1)
				go func(uid string, node *transport_pb.NodeInfo) {

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

func (m *NodesMap) ProbeNode(node *transport_pb.NodeInfo) error {

	// todo

	return nil
}

func (m *NodesMap) StartRecoveryProcess(node *transport_pb.NodeInfo) error {

	// todo

	return nil
}

func (m *NodesMap) NotifyNoAliveNodes() <-chan struct{} {
	return m.noAliveNodesCh
}
