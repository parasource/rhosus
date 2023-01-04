package cluster

import (
	"context"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/rs/zerolog/log"
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
		return nil, fmt.Errorf("error listening tcp: %v", err)
	}

	grpcServer := grpc.NewServer()
	control_pb.RegisterControlServer(grpcServer, s)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("error starting grpc control server")
		}
	}()

	go func() {
		select {
		case <-s.cluster.NotifyShutdown():
			err := lis.Close()
			if err != nil {
				log.Error().Err(err).Msg("error closing control server tcp")
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
		log.Debug().Str("candidate_id", req.CandidateId).Msg("declined request vote from a node with less term")
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// If we already voted in this term - we decline
	if s.lastVotedTerm == req.Term {
		log.Debug().Msg("declined request vote: already voted")
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	if req.LastLogIndex < s.cluster.lastLogIndex {
		log.Debug().Str("candidate_id", req.CandidateId).Msg("rejected candidate with outdated logs")
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	if req.LastLogTerm < s.cluster.lastLogTerm {
		log.Debug().Str("candidate_id", req.CandidateId).Uint32("last_log_term", req.LastLogTerm)
		return &control_pb.RequestVoteResponse{
			From:        s.cluster.ID,
			Term:        s.cluster.currentTerm,
			VoteGranted: false,
		}, nil
	}

	if req.LastLogTerm == s.cluster.lastLogTerm && req.LastLogIndex < s.cluster.lastLogIndex {
		log.Debug().Str("candidate_id", req.CandidateId).Msg("rejected candidate with short logs")
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
	currentTerm := s.cluster.GetCurrentTerm()
	lastLogIndex := s.cluster.GetLastLogIndex()

	if req.Term > currentTerm || (len(req.Entries) > 0 && req.Entries[len(req.Entries)-1].Index > lastLogIndex) {
		s.cluster.becomeFollower()
		s.cluster.SetCurrentTerm(req.Term)
		s.votedFor = ""
	}

	if req.Term < currentTerm {
		log.Debug().Str("leader_id", req.LeaderId).Msg("declined append entries from leader with less term")
		return &control_pb.AppendEntriesResponse{
			From:               s.cluster.ID,
			Term:               s.cluster.currentTerm,
			Success:            false,
			LastAgreedIndex:    s.cluster.lastLogIndex,
			LastCommittedIndex: s.cluster.lastLogIndex,
		}, nil
	}

	s.cluster.entriesAppendedC <- struct{}{}

	// Consistency is broken, so something below happened:
	// - current node was down for some time, and we need
	//   to recover it by loading all the missing entries
	// - something else
	if req.PrevLogIndex > int64(lastLogIndex) {
		return &control_pb.AppendEntriesResponse{
			From:               s.cluster.ID,
			Term:               currentTerm,
			Success:            false,
			LastCommittedIndex: lastLogIndex,
		}, nil
	}

	err := s.cluster.WriteEntriesFromLeader(req.Entries)
	if err != nil {
		log.Error().Err(err).Msg("error writing leader entries to wal")
		return &control_pb.AppendEntriesResponse{
			From:               s.cluster.ID,
			Term:               s.cluster.GetCurrentTerm(),
			Success:            false,
			LastCommittedIndex: s.cluster.GetLastLogIndex(),
		}, nil
	}

	return &control_pb.AppendEntriesResponse{
		From:               s.cluster.ID,
		Term:               s.cluster.GetCurrentTerm(),
		Success:            true,
		LastCommittedIndex: s.cluster.GetLastLogIndex(),
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
