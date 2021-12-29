/*
 * Copyright (c) 2021.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package registry

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

type NodeServerConfig struct {
	Host     string
	Port     string
	Password string
}

type NodeServer struct {
	registry *Registry

	Config NodeServerConfig
	server *grpc.Server

	shutdownCh chan struct{}
	readyCh    chan struct{}

	registerNodeFun func()
}

func NewNodeServer(config NodeServerConfig, registry *Registry) (*NodeServer, error) {
	var err error

	server := &NodeServer{
		Config:   config,
		registry: registry,

		shutdownCh: make(chan struct{}, 1),
		readyCh:    make(chan struct{}, 1),
	}

	grpcServer := grpc.NewServer()

	server.server = grpcServer

	return server, err
}

func (s *NodeServer) Run() error {

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

func (s *NodeServer) NotifyReady() <-chan struct{} {
	return s.readyCh
}
