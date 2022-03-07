package transport

import (
	"context"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"sync"
)

type Config struct {
	WriteTimeoutMs int
}

type Transport struct {
	config Config

	mu    sync.RWMutex
	conns map[string]*transport_pb.TransportServiceClient

	shutdown  bool
	shutdownC chan struct{}
}

func NewTransport(config Config, conns map[string]*transport_pb.TransportServiceClient) *Transport {

	t := &Transport{
		config: config,
		conns:  make(map[string]*transport_pb.TransportServiceClient),

		shutdownC: make(chan struct{}),
	}

	for id, connR := range conns {
		conn := *connR
		if _, err := conn.Heartbeat(context.Background(), &transport_pb.HeartbeatRequest{}); err != nil {
			logrus.Errorf("error adding conn to transport: %v", err)
			continue
		}
		t.conns[id] = &conn
	}

	return t
}

func (t *Transport) AddConn(id string, conn *transport_pb.TransportServiceClient) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.shutdown {
		return
	}
	if _, ok := t.conns[id]; ok {
		return
	}

	t.conns[id] = conn
}

func (t *Transport) RemoveConn(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.shutdown {
		return
	}
	if _, ok := t.conns[id]; !ok {
		return
	}

	delete(t.conns, id)
}

func (t *Transport) Shutdown() {
	t.mu.Lock()
	t.shutdown = true
	t.mu.Unlock()

	close(t.shutdownC)
}

func (t *Transport) NotifyShutdown() <-chan struct{} {
	return t.shutdownC
}
