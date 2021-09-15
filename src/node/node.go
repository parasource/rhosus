package rhosus_node

import (
	"parasource/rhosus/src/logging"
	"sync"
	"time"
)

type Config struct {
	Name    string
	Address string
	Timeout time.Duration
	Dir     []string
	Logger  *logging.LogHandler
}

type Node struct {
	cfg    Config
	logger *logging.LogHandler

	mu       sync.RWMutex
	shutdown chan struct{}
}

func NewNode(config Config) *Node {
	return &Node{
		cfg:    config,
		logger: config.Logger,

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
				n.logger.Log(logging.NewLogEntry(logging.LogLevelInfo, "ticker"))
			}
		}
	}()

	return nil
}

func (n *Node) Shutdown() {
	n.shutdown <- struct{}{}
}
