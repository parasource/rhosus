package node_server

import (
	"context"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

type ServerConfig struct {
	Host     string
	Port     string
	Password string
}

type Server struct {
	node_pb.NodeServiceServer

	Config ServerConfig
	server *grpc.Server

	shutdownCh chan struct{}
	readyCh    chan struct{}
}

func NewNodeServer(config ServerConfig) (*Server, error) {
	var err error

	server := &Server{
		Config: config,

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}, 1),
	}

	grpcServer := grpc.NewServer()
	node_pb.RegisterNodeServiceServer(grpcServer, server)

	server.server = grpcServer

	return server, err
}

func (s *Server) Run() error {

	address := net.JoinHostPort(s.Config.Host, s.Config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go func() {
		if err := s.server.Serve(lis); err != nil {
			logrus.Fatalf("error starting grpc node server: %v", err)
		}
	}()

	s.readyCh <- struct{}{}
	logrus.Infof("node service server successfully started on localhost:6435")

	return nil

}

func (s *Server) Ping(ctx context.Context, req *node_pb.PingRequest) (*node_pb.PingResponse, error) {
	logrus.Infof("PING")
	return &node_pb.PingResponse{}, nil
}

func (s *Server) NotifyShutdown() <-chan struct{} {
	return s.shutdownCh
}

func (s *Server) NotifyReady() <-chan struct{} {
	return s.readyCh
}
