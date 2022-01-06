package registry

import (
	"context"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/watcher"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sync"
)

type ControlClient struct {
	control_pb.ControlClient

	mu       sync.RWMutex
	registry *Registry
	servers  map[string]*control_pb.ControlClient

	// Watcher watches other registries condition etc.
	watcher *watcher.Watcher
}

type ServerAddress struct {
	Host     string
	Port     string
	Password string
}

type errorsBuffer []error

func NewControlClient(registry *Registry, addresses map[string]ServerAddress) (*ControlClient, error) {
	servers := make(map[string]*control_pb.ControlClient, len(addresses))
	errors := make(errorsBuffer, len(addresses))

	w := &watcher.Watcher{}
	go w.Watch()

	client := &ControlClient{
		registry: registry,
		servers:  make(map[string]*control_pb.ControlClient),

		watcher: w,
	}

	for uid, address := range addresses {
		address := net.JoinHostPort(address.Host, address.Port)

		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		// ping new node
		c := control_pb.NewControlClient(conn)
		_, err = c.Alive(context.Background(), &control_pb.Void{})
		if err != nil {
			// TODO: write something more informative
			logrus.Errorf("error connecting to registry: %v", err)
			errors = append(errors, err)
			conn.Close()
			continue
		}

		servers[uid] = &c
	}

	// No other registries are alive
	if len(errors) == len(addresses) {
		// TODO: this is actually important
	}

	client.mu.Lock()
	for uid, server := range servers {
		client.servers[uid] = server
	}
	client.mu.Unlock()

	return client, nil
}

func (c *ControlClient) InitVotingProcess() {

}
