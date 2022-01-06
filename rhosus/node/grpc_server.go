package rhosus_node

import (
	"context"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

type GrpcServerConfig struct {
	Host     string
	Port     string
	Password string
}

type GrpcServer struct {
	transport_pb.TransportServiceServer

	node *Node

	Config GrpcServerConfig
	server *grpc.Server

	shutdownCh chan struct{}
	readyCh    chan struct{}

	registerNodeFun func()
}

func NewGrpcServer(config GrpcServerConfig, node *Node) (*GrpcServer, error) {
	var err error

	server := &GrpcServer{
		Config: config,
		node:   node,

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}, 1),
	}

	return server, err
}

func (s *GrpcServer) Ping(c context.Context, r *transport_pb.PingRequest) (*transport_pb.PingResponse, error) {
	return &transport_pb.PingResponse{}, nil
}

func (s *GrpcServer) ShutdownNode(c context.Context, r *transport_pb.ShutdownNodeRequest) (*transport_pb.ShutdownNodeResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) AssignBlocks(c context.Context, r *transport_pb.AssignBlocksRequest) (*transport_pb.AssignBlocksResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) RemoveBlocks(c context.Context, r *transport_pb.RemoveBlocksRequest) (*transport_pb.RemoveBlocksResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) PlacePages(c context.Context, r *transport_pb.PlacePagesRequest) (*transport_pb.PlacePagesResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) FetchMetrics(c context.Context, r *transport_pb.FetchMetricsRequest) (*transport_pb.FetchMetricsResponse, error) {
	metrics, err := s.node.CollectMetrics()
	if err != nil {
		logrus.Errorf("error fetching metrics: %v", err)
		return nil, err
	}

	return &transport_pb.FetchMetricsResponse{
		Name:    s.node.Name,
		Metrics: metrics,
	}, nil
}

func (s *GrpcServer) Run() {

	address := net.JoinHostPort(s.Config.Host, s.Config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalf("error listening tcp: %v", err)
	}

	grpcServer := grpc.NewServer()
	transport_pb.RegisterTransportServiceServer(grpcServer, s)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logrus.Fatalf("error starting grpc node server: %v", err)
		}
	}()

	s.readyCh <- struct{}{}
	logrus.Infof("node service server successfully started on %v", address)

	for {
		select {
		case <-s.NotifyShutdown():
			err := lis.Close()
			if err != nil {
				logrus.Errorf("error closing tcp connection: %v", err)
			}
		}
	}
}

func (s *GrpcServer) NotifyShutdown() <-chan struct{} {
	return s.shutdownCh
}

func (s *GrpcServer) NotifyReady() <-chan struct{} {
	return s.readyCh
}
