package registry

import (
	"context"
	"errors"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
	"github.com/parasource/rhosus/rhosus/registry/recovery"
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

	RecoveryManager *recovery.Manager
}

type NodeInfo transmission_pb.NodeInfo
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

func (m *NodesMap) AddNode(uid string, info *transmission_pb.NodeInfo) error {

	address := net.JoinHostPort(info.Address.Host, info.Address.Port)
	logrus.Infof("connecting to grpc: %v", address)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}

	// ping new node
	client := transmission_pb.NewTransmissionServiceClient(conn)
	_, err = client.Ping(context.Background(), &transmission_pb.PingRequest{})
	if err != nil {
		return err
	}

	m.mu.Lock()
	if _, ok := m.nodes[uid]; !ok {
		m.nodes[uid] = (*NodeInfo)(info)
	}
	if _, ok := m.nodesConns[uid]; !ok {
		m.nodesConns[uid] = (*NodeGrpcConn)(conn)
	}
	m.nodesProbeTries[uid] = 0
	m.mu.Unlock()

	return nil
}

func (m *NodesMap) RemoveNode(uid string) {
	m.mu.Lock()
	delete(m.nodes, uid)
	delete(m.nodesProbeTries, uid)
	if _, ok := m.nodesConns[uid]; ok {
		delete(m.nodesProbeTries, uid)
	}
	m.mu.Unlock()
}

func (m *NodesMap) NodeExists(uid string) bool {
	m.mu.RLock()
	_, ok := m.nodes[uid]
	m.mu.RUnlock()

	return ok
}

func (m *NodesMap) GetGrpcClient(uid string) (transmission_pb.TransmissionServiceClient, error) {
	m.mu.RLock()
	conn, ok := m.nodesConns[uid]
	m.mu.RUnlock()

	if !ok {
		return nil, errors.New("undefined uid. node does not persist in node map")
	}

	client := transmission_pb.NewTransmissionServiceClient((*grpc.ClientConn)(conn))
	return client, nil
}

func (m *NodesMap) WatchNodes() {

	ticker := tickers.SetTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:
			nodes := make(map[string]*transmission_pb.NodeInfo, len(m.nodes))

			m.mu.RLock()
			for uid, info := range m.nodes {
				nodes[uid] = (*transmission_pb.NodeInfo)(info)
			}
			m.mu.RUnlock()

			aliveNodes := 0
			wg := sync.WaitGroup{}

			for uid, node := range nodes {
				wg.Add(1)
				go func(uid string, node *transmission_pb.NodeInfo) {

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

func (m *NodesMap) ProbeNode(node *transmission_pb.NodeInfo) error {

	// todo

	return nil
}

func (m *NodesMap) StartRecoveryProcess(node *transmission_pb.NodeInfo) error {

	// todo

	return nil
}

func (m *NodesMap) NotifyNoAliveNodes() <-chan struct{} {
	return m.noAliveNodesCh
}
