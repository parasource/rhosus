package rhosus_node

import (
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
