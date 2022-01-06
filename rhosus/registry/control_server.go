package registry

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
	control_pb.ControlServer

	Config ControlServerConfig

	shutdownCh chan struct{}
}

func NewControlServer() (*ControlServer, error) {
	var err error

	s := &ControlServer{}

	address := net.JoinHostPort(s.Config.Host, s.Config.Port)
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
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Fatalf("error listening grpc node server: %v", err)
		}
	}()
	go func() {
		for {
			select {
			case <-s.NotifyShutdown():
				err := lis.Close()
				if err != nil {
					logrus.Errorf("error closing control server tcp: %v", err)
				}

				return
			}
		}
	}()

	return s, err
}

func (s *ControlServer) RequestVote(c context.Context, req *control_pb.RequestVoteRequest) (*control_pb.RequestVoteResponse, error) {
	panic("implement me")
}

func (s *ControlServer) AppendEntries(c context.Context, req *control_pb.AppendEntriesRequest) (*control_pb.AppendEntriesResponse, error) {
	panic("implement me")
}

func (s *ControlServer) Shutdown(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Alive(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Online(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) Offline(c context.Context, req *control_pb.Void) (*control_pb.Void, error) {
	panic("implement me")
}

func (s *ControlServer) NotifyShutdown() <-chan struct{} {
	return s.shutdownCh
}
