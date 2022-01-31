package cluster

import (
	"context"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

type ControlServerConfig struct {
	Host     string
	Port     string
	Password string
}

type ControlServer struct {
	cluster *Cluster
	control_pb.ControlServer

	votedFor string

	Config ControlServerConfig
}

func NewControlServer(cluster *Cluster, address string) (*ControlServer, error) {
	var err error

	s := &ControlServer{
		cluster: cluster,
	}

	lis, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalf("error listening tcp: %v", err)
	}

	grpcServer := grpc.NewServer()
	control_pb.RegisterControlServer(grpcServer, s)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Fatalf("error starting grpc node server: %v", err)
		}
	}()

	go func() {
		select {
		case <-s.cluster.NotifyShutdown():
			err := lis.Close()
			if err != nil {
				logrus.Errorf("error closing control server tcp: %v", err)
			}

			return
		}
	}()

	return s, err
}

func (s *ControlServer) RequestVote(c context.Context, req *control_pb.RequestVoteRequest) (*control_pb.RequestVoteResponse, error) {

	// If the term of the candidate is less than our - we don't grant a vote
	if req.Term <= s.cluster.term {
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.term,
			VoteGranted: false,
		}, nil
	}

	if req.Term > s.cluster.term {
		// step down
	}

	cond1 := s.votedFor == ""
	cond2 := s.votedFor == req.CandidateUid
	cond3 := req.LastLogIndex >= s.cluster.lastLogIndex-uint64(1)

	if (cond1 || cond2) && cond3 {
		s.votedFor = req.CandidateUid
		// set new term
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        req.Term,
			VoteGranted: true,
		}, nil
	}

	return &control_pb.RequestVoteResponse{
		From:        s.cluster.ID,
		Term:        s.cluster.term,
		VoteGranted: false,
	}, nil
}

func (s *ControlServer) AppendEntries(c context.Context, req *control_pb.AppendEntriesRequest) (*control_pb.AppendEntriesResponse, error) {
	panic("implement me")
}

func (s *ControlServer) Shutdown(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Alive(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	logrus.Warnf("IM ALIVE")
	return &control_pb.Void{}, nil
}

func (s *ControlServer) Online(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Offline(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}
