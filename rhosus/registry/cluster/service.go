package cluster

import (
	"context"
	"errors"
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
	conns   map[string]*PeerConn
	// uid of the leader peer
	currentLeader string
}

type PeerConn struct {
	Uid   string
	Alive bool

	conn *control_pb.ControlClient
}

type errorsBuffer []error

func NewControlService(cluster *Cluster, addresses map[string]string) (*ControlService, []string, error) {

	peers := make(map[string]*control_pb.ControlClient, len(addresses))
	errors := make(errorsBuffer, len(addresses))

	service := &ControlService{
		conns:   make(map[string]*PeerConn),
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
		service.conns[uid] = &PeerConn{
			conn:  conn,
			Alive: true,
		}
	}

	return service, sanePeers, nil
}

func (s *ControlService) AppendEntries(req *control_pb.AppendEntriesRequest) map[string]*control_pb.AppendEntriesResponse {

	resp := make(map[string]*control_pb.AppendEntriesResponse)

	for uid, peer := range s.conns {
		go func(uid string, peer *PeerConn) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
			defer cancel()

			conn := *peer.conn

			_, err := conn.AppendEntries(ctx, req)
			if err != nil {
				logrus.Errorf("error sending entries to peer %v: %v", uid, err)
			}

			//resp[uid] = res
		}(uid, peer)
	}

	return resp
}

func (s *ControlService) AddPeer(uid string, address string) error {

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		conn.Close()
		return err
	}

	// ping new node
	c := control_pb.NewControlClient(conn)

	start := time.Now()
	_, err = c.Alive(context.Background(), &control_pb.Void{})
	if err != nil {
		conn.Close()
		return err
	}
	logrus.Infof("time passed: %v", time.Since(start).String())

	s.conns[uid] = &PeerConn{
		conn: &c,
	}

	return nil
}

// We actually don't need this function because
// either way we mark unavailable conns
func (s *ControlService) removePeer(uid string) error {
	return nil
}

// getCurrentLeader returns current cluster leader
func (s *ControlService) getCurrentLeader() *PeerConn {

	return s.conns[s.currentLeader]
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

func (s *ControlService) sendVoteRequest(uid string) (*control_pb.RequestVoteResponse, error) {

	if peer, ok := s.conns[uid]; ok {

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		conn := *peer.conn
		return conn.RequestVote(ctx, &control_pb.RequestVoteRequest{
			Term:         s.cluster.currentTerm,
			CandidateUid: s.cluster.ID,
		})
	}

	return nil, errors.New("conn not found")
}

func (s *ControlService) sendVoteRequests() chan voteResponse {

	peers := make(map[string]*PeerConn, len(s.conns))
	for uid, peer := range s.conns {
		peers[uid] = peer
	}

	respC := make(chan voteResponse, len(peers))

	for uid, peer := range peers {

		go func(uid string, peer *PeerConn) {
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
				return
			}

			respC <- voteResponse{
				res: res,
				err: nil,
			}

			logrus.Infof("response from peer %v: %v", res.From, res.VoteGranted)
		}(uid, peer)
	}

	return respC
}
