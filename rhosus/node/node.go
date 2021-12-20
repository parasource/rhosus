package rhosus_node

import (
	"context"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type Config struct {
	Name    string
	Address string
	Timeout time.Duration
	Dir     []string
}

type Node struct {
	cfg Config

	mu       sync.RWMutex
	shutdown chan struct{}
}

func NewNode(config Config) *Node {
	return &Node{
		cfg: config,

		shutdown: make(chan struct{}, 1),
	}
}

func (n *Node) Start() error {

	ticker := time.NewTicker(time.Second)

	go func() {
		for {
			select {
			case <-n.shutdown:
				return
			case <-ticker.C:
				conn, err := grpc.Dial("localhost:6435", grpc.WithInsecure())
				if err != nil {
					logrus.Errorf("error dialing node server: %v", err)
					conn.Close()
					continue
				}

				client := node_pb.NewNodeServiceClient(conn)

				_, err = client.Ping(context.Background(), &node_pb.PingRequest{})
				if err != nil {
					logrus.Errorf("error sending ping commmand: %v", err)
					conn.Close()
					continue
				}

				logrus.Infof("PONG")

				conn.Close()
			}
		}
	}()

	return nil
}

func (n *Node) NotifyShutdown() <-chan struct{} {
	return n.shutdown
}

func (n *Node) Shutdown() {
	n.shutdown <- struct{}{}
}
