package rhosus_node

import (
	"context"
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
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
	transmission_pb.TransmissionServiceServer

	node *Node

	Config     GrpcServerConfig
	server     *grpc.Server
	etcdClient *rhosus_etcd.EtcdClient

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

	grpcServer := grpc.NewServer()
	transmission_pb.RegisterTransmissionServiceServer(grpcServer, server)

	server.server = grpcServer

	return server, err
}

func (s *GrpcServer) ShutdownNode(c context.Context, r *transmission_pb.ShutdownNodeRequest) (*transmission_pb.ShutdownNodeResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) AssignBlocks(c context.Context, r *transmission_pb.AssignBlocksRequest) (*transmission_pb.AssignBlocksResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) RemoveBlocks(c context.Context, r *transmission_pb.RemoveBlocksRequest) (*transmission_pb.RemoveBlocksResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) PlacePages(c context.Context, r *transmission_pb.PlacePagesRequest) (*transmission_pb.PlacePagesResponse, error) {
	panic("implement me")
}

func (s *GrpcServer) Run() {

	address := net.JoinHostPort(s.Config.Host, s.Config.Port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalf("error listening tcp")
	}

	go func() {
		if err := s.server.Serve(lis); err != nil {
			logrus.Fatalf("error starting grpc node server: %v", err)
		}
	}()

	s.readyCh <- struct{}{}
	logrus.Infof("node service server successfully started on localhost:6435")

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
