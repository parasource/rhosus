package registry

import (
	"context"
	"errors"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/registry/transport"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("node is not found")
)

type NodesMap struct {
	registry *Registry

	mu        sync.RWMutex
	nodes     map[string]*Node
	transport *transport.Transport

	// max number of ping retries, before the node is marked as unavailable
	pingIntervalMs int
	maxPingRetries int
}

type Node struct {
	info    *transport_pb.NodeInfo
	metrics *transport_pb.NodeMetrics

	conn         *transport_pb.TransportServiceClient
	mu           sync.RWMutex
	pingRetries  int
	lastActivity time.Time
	recovering   bool
	unavailable  bool
}

func NewNodesMap(registry *Registry, nodes map[string]*transport_pb.NodeInfo) (*NodesMap, error) {
	n := &NodesMap{
		registry: registry,
		nodes:    make(map[string]*Node),

		pingIntervalMs: 500,
		maxPingRetries: 3,
	}

	for id, info := range nodes {
		address := net.JoinHostPort(info.Address.Host, info.Address.Port)

		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			logrus.Errorf("error connnecting to node %v: %v", info.Id, err)
			continue
		}

		// ping new node
		client := transport_pb.NewTransportServiceClient(conn)
		_, err = client.Ping(context.Background(), &transport_pb.PingRequest{})
		if err != nil {
			logrus.Errorf("error pinging node %v: %v", info.Id, err)
			continue
		}

		n.nodes[id] = &Node{
			info:         info,
			conn:         &client,
			lastActivity: time.Now(),
			recovering:   false,
		}

		logrus.Infof("added existing node %v on %v", info.Id, address)
	}

	conns := make(map[string]*transport_pb.TransportServiceClient)
	for id, node := range n.nodes {
		conns[id] = node.conn
	}
	t := transport.NewTransport(transport.Config{
		WriteTimeoutMs: 800,
	}, conns)
	n.transport = t

	return n, nil
}

func (m *NodesMap) AddNode(name string, info *transport_pb.NodeInfo) error {

	address := net.JoinHostPort(info.Address.Host, info.Address.Port)

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
	m.nodes[name] = &Node{
		info: info,

		conn:         &client,
		lastActivity: time.Now(),
		recovering:   false,
	}
	m.mu.Unlock()

	logrus.Infof("added new node to nodes map: %v %v", info.Id, address)

	return nil
}

// RemoveNode is called only when node is gracefully shut down
// otherwise it is just marked as temporarily unavailable
func (m *NodesMap) RemoveNode(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.nodes, id)
}

func (m *NodesMap) UpdateNodeInfo(id string, info *transport_pb.NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node, ok := m.nodes[id]; ok {
		node.info = info
	}
}

func (m *NodesMap) NodeExists(name string) bool {
	m.mu.RLock()
	_, ok := m.nodes[name]
	m.mu.RUnlock()

	return ok
}

func (m *NodesMap) WatchNodes() {

	ticker := tickers.SetTicker(time.Millisecond * time.Duration(m.pingIntervalMs))

	for {
		select {
		case <-m.registry.NotifyShutdown():
			return
		case <-ticker.C:
			nodes := make(map[string]*Node, len(m.nodes))

			m.mu.RLock()
			for uid, node := range m.nodes {
				nodes[uid] = node
			}
			m.mu.RUnlock()

			aliveNodes := 0
			wg := sync.WaitGroup{}

			for uid, node := range nodes {
				wg.Add(1)
				go func(id string, node *Node) {
					defer wg.Done()
					conn := *node.conn

					_, err := conn.Ping(context.Background(), &transport_pb.PingRequest{})
					if err != nil {
						// Node does not respond to health probes

						m.mu.RLock()
						tries := m.nodes[id].pingRetries
						m.mu.RUnlock()

						if tries >= m.maxPingRetries {
							m.mu.Lock()
							m.nodes[id].unavailable = true
							m.mu.Unlock()
						}

						m.mu.Lock()
						m.nodes[id].pingRetries++
						m.mu.Unlock()

						return
					}

					m.mu.Lock()
					aliveNodes++
					m.mu.Unlock()

				}(uid, node)
			}

			wg.Wait()

			if aliveNodes <= 1 {
				// todo
			}

		}
	}
}

func (m *NodesMap) GetNode(id string) *Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getNode(id)
}

func (m *NodesMap) getNode(id string) *Node {
	return m.nodes[id]
}

func (m *NodesMap) AssignBlocks(nodeID string, blocks []*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {
	node := m.GetNode(nodeID)
	if node == nil {
		return nil, ErrNotFound
	}

	stream, err := (*node.conn).AssignBlocks(context.Background())
	if err != nil {
		return nil, err
	}

	// Sending blocks
	for _, block := range blocks {
		err := stream.Send(&transport_pb.AssignBlockRequest{
			Block: block,
		})
		if err != nil {
			logrus.Errorf("error sending block: %v", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		logrus.Errorf("lksjdflkjs: %v", err)
		return nil, err
	}

	return res.Placement, nil
}

func (m *NodesMap) StartRecoveryProcess(node *transport_pb.NodeInfo) error {

	// todo

	return nil
}
