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

	lastVotedTerm uint32
	votedFor      string

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

	currentTerm := s.cluster.GetCurrentTerm()

	if req.Term > currentTerm {
		s.cluster.SetCurrentTerm(req.Term)
		s.cluster.becomeFollower()
		s.votedFor = ""
	}

	if req.Term < currentTerm {
		logrus.Warnf("DECLINED LESS TERM: %v --- %v", req, s.cluster.currentTerm)
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// If we already voted in this term - we decline
	if s.lastVotedTerm == req.Term {

		logrus.Warnf("DECLINED ALREADY VOTED: %v --- %v", req, s.cluster.currentTerm)
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	if req.LastLogIndex < s.cluster.lastLogIndex {

		logrus.Warnf("DECLINED LESS LOG INDEX: %v --- %v", req.LastLogIndex, s.cluster.lastLogIndex)
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	s.cluster.SetCurrentTerm(req.Term)
	s.lastVotedTerm = req.Term

	return &control_pb.RequestVoteResponse{
		From:        s.cluster.ID,
		Term:        currentTerm,
		VoteGranted: true,
	}, nil
}

func (s *ControlServer) AppendEntries(c context.Context, req *control_pb.AppendEntriesRequest) (*control_pb.AppendEntriesResponse, error) {

	logrus.Infof("HEARTBEAT TERM: %v", req.Term)

	currentTerm := s.cluster.GetCurrentTerm()

	logrus.Warnf("TERMS: %v -- %v", req.Term, currentTerm)
	if req.Term > currentTerm {
		s.cluster.becomeFollower()
		s.cluster.SetCurrentTerm(req.Term)
		s.votedFor = ""
	}

	if req.Term < currentTerm {
		logrus.Warnf("DECLINED LESS TERM: %v -- %v", req.Term, currentTerm)
		return &control_pb.AppendEntriesResponse{
			From:               s.cluster.ID,
			Term:               s.cluster.currentTerm,
			Success:            false,
			LastAgreedIndex:    s.cluster.lastLogIndex,
			LastCommittedIndex: s.cluster.lastLogIndex,
		}, nil
	}

	s.cluster.entriesAppendedC <- struct{}{}

	// If there are no entries then it is used just for the heartbeat
	if len(req.Entries) < 1 {

		return &control_pb.AppendEntriesResponse{
			From:               s.cluster.ID,
			Term:               s.cluster.currentTerm,
			Success:            true,
			LastAgreedIndex:    s.cluster.lastLogIndex,
			LastCommittedIndex: s.cluster.lastLogIndex,
		}, nil
	}

	logrus.Warnf("DELTA: %v -- %v", req.Entries[0].Index, s.cluster.lastLogIndex)
	// Consistency is broken, so something below happened:
	// - current node was down for some time, and we need
	//   to recover it by loading all the missing entries
	// - something else
	if req.Entries[0].Index-s.cluster.lastLogIndex != 1 {

		return &control_pb.AppendEntriesResponse{
			From:               s.cluster.ID,
			Term:               s.cluster.currentTerm,
			Success:            false,
			LastCommittedIndex: s.cluster.lastLogIndex,
		}, nil
	}

	err := s.cluster.WriteEntriesFromLeader(req.Entries)
	if err != nil {
		logrus.Errorf("error writing to wal: %v", err)
	}

	return &control_pb.AppendEntriesResponse{
		From:               s.cluster.ID,
		Term:               s.cluster.currentTerm,
		Success:            true,
		LastCommittedIndex: s.cluster.lastLogIndex,
	}, nil
}

func (s *ControlServer) Shutdown(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Alive(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	return &control_pb.Void{}, nil
}

func (s *ControlServer) Online(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Offline(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}
