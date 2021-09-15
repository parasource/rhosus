package rhosus_node

import (
	"parasource/rhosus/rhosus/rlog"
	"sync"
	"time"
)

type Config struct {
	Name    string
	Address string
	Timeout time.Duration
	Dir     []string
	Logger  *rlog.LogHandler
}

type Node struct {
	cfg    Config
	Logger *rlog.LogHandler

	mu       sync.RWMutex
	shutdown chan struct{}
}

func NewNode(config Config) *Node {
	return &Node{
		cfg:    config,
		Logger: config.Logger,

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
				n.Logger.Log(rlog.NewLogEntry(rlog.LogLevelInfo, "ticker"))
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
