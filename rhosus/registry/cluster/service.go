package cluster

import (
	"context"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

type ServerAddress struct {
	Host     string
	Port     string
	Username string
	Password string
}

type ControlService struct {
	cluster *Cluster
	peers   map[string]*Peer
	// uid of the leader peer
	currentLeader string
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

func NewControlService(cluster *Cluster, addresses map[string]string) (*ControlService, []string, error) {
	peers := make(map[string]*control_pb.ControlClient, len(addresses))
	errors := make(errorsBuffer, len(addresses))

	service := &ControlService{
		peers:   make(map[string]*Peer),
		cluster: cluster,
	}

	var sanePeers []string
	for uid, address := range addresses {

		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			logrus.Errorf("couldn't connect to a registry peer: %v", err)
			continue
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
		sanePeers = append(sanePeers, uid)

		logrus.Infof("connected to peer %v", address)
	}

	// No other registries are alive
	if len(errors) == len(addresses) {
		// TODO: this is actually important
	}

	for uid, conn := range peers {
		service.peers[uid] = &Peer{
			conn:  conn,
			Alive: true,
		}
	}

	return service, sanePeers, nil
}

func (s *ControlService) Start() {

	//for {
	//	select {
	//	case <-s.observer.NotifyStartVoting():
	//
	//		responses := s.sendVoteRequests()
	//		for _, res := range responses {
	//
	//			// Here we check if peer replied to our voteRequest
	//			if res.isSuccessful() {
	//				// TODO
	//			} else {
	//
	//			}
	//		}
	//
	//	}
	//}

}

func (s *ControlService) AppendEntries(req *control_pb.AppendEntriesRequest) map[string]*control_pb.AppendEntriesResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp := make(map[string]*control_pb.AppendEntriesResponse)

	for uid, peer := range s.peers {
		conn := *peer.conn

		res, err := conn.AppendEntries(ctx, req)
		if err != nil {
			logrus.Errorf("error sending entries to peer %v: %v", uid, err)
		}

		resp[uid] = res
	}

	return resp
}

// isLeaderPresent checks if the leader already
// present in cluster
func (s *ControlService) isLeaderPresent() bool {

	for _, peer := range s.peers {
		if peer.IsLeader {
			return true
		}
	}

	return false
}

func (s *ControlService) AddPeer(uid string, address string, isLeader bool) error {

	conn, err := grpc.Dial(address, grpc.WithInsecure())
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

	s.peers[uid] = &Peer{
		conn:     &c,
		IsLeader: isLeader,
	}

	return nil
}

// We actually don't need this function because
// either way we mark unavailable peers
func (s *ControlService) removePeer(uid string) error {
	return nil
}

// getCurrentLeader returns current cluster leader
func (s *ControlService) getCurrentLeader() *Peer {

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

func (s *ControlService) sendVoteRequests() chan voteResponse {

	peers := make(map[string]*Peer, len(s.peers))
	for uid, peer := range s.peers {
		peers[uid] = peer
	}

	respC := make(chan voteResponse)

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
				respC <- voteResponse{
					res: nil,
					err: err,
				}
			}

			respC <- voteResponse{
				res: res,
				err: nil,
			}

			logrus.Infof("response from peer %v: %v", res.From, res)
		}(uid, peer)
	}

	return respC
}
