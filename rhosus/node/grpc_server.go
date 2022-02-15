package rhosus_node

import (
	"context"
	"errors"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
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

	shutdownC chan struct{}
	readyC    chan struct{}

	registerNodeFun func()
}

func NewGrpcServer(config GrpcServerConfig, node *Node) (*GrpcServer, error) {
	var err error

	server := &GrpcServer{
		Config: config,
		node:   node,

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

	return server, err
}

func (s *GrpcServer) Ping(c context.Context, r *transport_pb.PingRequest) (*transport_pb.PingResponse, error) {
	return &transport_pb.PingResponse{}, nil
}

func (s *GrpcServer) ShutdownNode(c context.Context, r *transport_pb.ShutdownNodeRequest) (*transport_pb.ShutdownNodeResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) AssignBlocks(srv transport_pb.TransportService_AssignBlocksServer) error {

	var blocks []*fs_pb.Block

	for {
		req, err := srv.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.Errorf("error receiving: %v", err)
			continue
		}

		blocks = append(blocks, req.Block)
	}

	if len(blocks) == 0 {
		return errors.New("empty blocks")
	}

	logrus.Infof("blocks len: %v", len(blocks))
	res, err := s.node.HandleAssignBlocks(blocks)
	if err != nil {
		return err
	}
	err = srv.SendAndClose(&transport_pb.AssignBlocksResponse{
		Placement: res,
	})
	if err != nil {
		logrus.Errorf("error sending and closing stream: %v", err)
	}

	return nil
}

func (s *GrpcServer) RemoveBlocks(c context.Context, r *transport_pb.RemoveBlocksRequest) (*transport_pb.RemoveBlocksResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) PlacePages(c context.Context, r *transport_pb.PlacePartitionRequest) (*transport_pb.PlacePartitionResponse, error) {
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

	s.readyC <- struct{}{}
	logrus.Infof("node service server successfully started on %v", address)

	if <-s.NotifyShutdown(); true {
		err := lis.Close()
		if err != nil {
			logrus.Errorf("error closing tcp connection: %v", err)
		}
	}
}

func (s *GrpcServer) NotifyShutdown() <-chan struct{} {
	return s.shutdownC
}

func (s *GrpcServer) NotifyReady() <-chan struct{} {
	return s.readyC
}
