package main

import (
	"context"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	"google.golang.org/grpc"
	"time"
)

func GetConn(address string) (*api_pb.ApiClient, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(32<<20), grpc.MaxCallRecvMsgSize(32<<20)))
	if err != nil {
		conn.Close()
		return nil, err
	}

	client := api_pb.NewApiClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = client.Ping(ctx, &api_pb.Void{})
	if err != nil {
		return nil, err
	}

	return &client, nil
}
