package registry

import (
	"context"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sync"
	"time"
)

type ServerAddress struct {
	Host     string
	Port     string
	Password string
}

type ControlService struct {
	control_pb.ControlClient

	mu       sync.RWMutex
	registry *Registry
	peers    map[string]*Peer
	// uid of the leader peer
	currentLeader string

	// Observer watches for heartbeats and notifies if leader is off
	observer *Observer
}

type Peer struct {
	Uid      string
	Alive    bool
	IsLeader bool

	conn *control_pb.ControlClient
}

func (p *Peer) isAlive() bool {
	return p.Alive
}

type errorsBuffer []error

func NewControlService(registry *Registry, addresses map[string]ServerAddress) (*ControlService, error) {
	peers := make(map[string]*control_pb.ControlClient, len(addresses))
	errors := make(errorsBuffer, len(addresses))

	service := &ControlService{
		registry: registry,
		peers:    make(map[string]*Peer),
	}

	observer := NewObserver(service)
	go observer.Observe()

	service.observer = observer

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

		peers[uid] = &c
	}

	// No other registries are alive
	if len(errors) == len(addresses) {
		// TODO: this is actually important
	}

	service.mu.Lock()
	for uid, conn := range peers {
		service.peers[uid] = &Peer{
			conn:  conn,
			Alive: true,
		}
	}
	service.mu.Unlock()

	return service, nil
}

func (s *ControlService) InitVoting() error {

	responses := s.sendVoteRequests()
	for _, res := range responses {

		// Here we check if peer replied to our voteRequest
		if res.isSuccessful() {
			// TODO
		} else {

		}
	}

	return nil
}

// isLeaderPresent checks if the leader already
// present in cluster
func (s *ControlService) isLeaderPresent() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, peer := range s.peers {
		if peer.IsLeader {
			return true
		}
	}

	return false
}

func (s *ControlService) AddPeer(uid string, address ServerAddress, isLeader bool) error {
	addr := net.JoinHostPort(address.Host, address.Port)

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		conn.Close()
		return err
	}

	// ping new node
	c := control_pb.NewControlClient(conn)
	_, err = c.Alive(context.Background(), &control_pb.Void{})
	if err != nil {
		conn.Close()
		return err
	}

	s.mu.Lock()
	s.peers[uid] = &Peer{
		conn:     &c,
		IsLeader: isLeader,
	}
	s.mu.Unlock()

	return nil
}

// We actually don't need this function because
// either way we mark unavailable peers
func (s *ControlService) removePeer(uid string) error {
	return nil
}

// getCurrentLeader returns current cluster leader
func (s *ControlService) getCurrentLeader() *Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.peers[s.currentLeader]
}

type voteResponse struct {
	res *control_pb.RequestVoteResponse
	err error
}

func (v voteResponse) isSuccessful() bool {
	return v.err == nil
}

func (v voteResponse) getError() error {
	return v.err
}

func (s *ControlService) sendVoteRequests() map[string]voteResponse {

	peers := make(map[string]*Peer, len(s.peers))
	s.mu.RLock()
	for uid, peer := range s.peers {
		peers[uid] = peer
	}
	s.mu.RUnlock()

	responses := make(map[string]voteResponse)

	for uid, peer := range peers {

		go func(uid string, peer *Peer) {
			conn := *peer.conn

			// TODO: move to configuration
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
			defer cancel()

			req := &control_pb.RequestVoteRequest{
				Term:         0,
				CandidateUid: "",
				LastLogIndex: 0,
				LastLogTerm:  0,
			}
			res, err := conn.RequestVote(ctx, req)
			if err != nil {
				// TODO: this is important too
				logrus.Errorf("error connecting to %v: %v", uid, err)
				responses[uid] = voteResponse{
					res: nil,
					err: err,
				}
			}

			responses[uid] = voteResponse{
				res: res,
				err: nil,
			}

			logrus.Infof("response from peer %v: %v", res.From, res)
		}(uid, peer)
	}

	return responses
}
