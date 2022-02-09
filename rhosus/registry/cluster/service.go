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
}

type PeerConn struct {
	Uid   string
	Alive bool

	conn *control_pb.ControlClient
}

type errorsBuffer []error

func NewControlService(cluster *Cluster, addresses map[string]string) (*ControlService, []string, error) {

	peers := make(map[string]*control_pb.ControlClient, len(addresses))
	errs := make(errorsBuffer, len(addresses))

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
			errs = append(errs, err)
			conn.Close()
			continue
		}

		peers[uid] = &c
		sanePeers = append(sanePeers, uid)

		logrus.Infof("connected to peer %v", address)
	}

	// No other registries are alive
	//if len(errs) == len(addresses) {
	//	service.cluster.becomeLeader()
	//}

	for uid, conn := range peers {
		service.conns[uid] = &PeerConn{
			conn:  conn,
			Alive: true,
		}
	}

	return service, sanePeers, nil
}

func (s *ControlService) AppendEntries(uid string, req *control_pb.AppendEntriesRequest) (*control_pb.AppendEntriesResponse, error) {

	if peer, ok := s.conns[uid]; ok {

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		conn := *peer.conn

		return conn.AppendEntries(ctx, req)
	}

	return nil, errors.New("peer not found")
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

func (s *ControlService) sendVoteRequest(uid string) (*control_pb.RequestVoteResponse, error) {

	if peer, ok := s.conns[uid]; ok {

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		conn := *peer.conn
		return conn.RequestVote(ctx, &control_pb.RequestVoteRequest{
			Term:         s.cluster.GetCurrentTerm(),
			LastLogIndex: s.cluster.lastLogIndex,
			CandidateId:  s.cluster.ID,
		})
	}

	return nil, errors.New("conn not found")
}
