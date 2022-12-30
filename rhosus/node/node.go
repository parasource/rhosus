/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package rhosus_node

import (
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	"github.com/parasource/rhosus/rhosus/node/data"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/profiler"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"
)

type Config struct {
	Name        string
	EtcdAddress string
	Address     string
	Timeout     time.Duration
	RhosusPath  string
}

type Node struct {
	ID     string
	Name   string
	Config Config

	data     *data.Manager
	stats    *StatsManager
	profiler *profiler.Profiler
	server   *GrpcServer
	etcd     *rhosus_etcd.EtcdClient

	shutdownC chan struct{}
	readyC    chan struct{}
}

func NewNode(config Config) (*Node, error) {

	v4uuid, _ := uuid.NewV4()

	node := &Node{
		ID:     v4uuid.String(),
		Name:   util.GenerateRandomName(3),
		Config: config,

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

	dataPath := path.Join(config.RhosusPath, "data")
	dataManager, err := data.NewManager(dataPath)
	if err != nil {
		logrus.Fatalf("error creating data manager: %v", err)
	}
	node.data = dataManager

	statsManager := NewStatsManager(node)
	node.stats = statsManager

	nodeProfiler, err := profiler.NewProfiler()
	if err != nil {
		logrus.Fatalf("error creating profiler: %v", err)
	}
	node.profiler = nodeProfiler

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Address: config.EtcdAddress,
		Timeout: 5,
	})
	if err != nil {
		logrus.Fatalf("error connecting to etcd: %v", err)
	}
	node.etcd = etcdClient

	grpcServer, err := NewGrpcServer(GrpcServerConfig{
		Address: config.Address,
	}, node)
	if err != nil {
		logrus.Errorf("error creating node grpc server: %v", err)
	}
	node.server = grpcServer

	return node, nil
}

func (n *Node) CollectMetrics() (*transport_pb.NodeMetrics, error) {

	v, err := n.profiler.GetMem()
	if err != nil {
		logrus.Errorf("error getting memory stats: %v", err)
	}

	usage := n.profiler.GetPathDiskUsage("/")

	metrics := &transport_pb.NodeMetrics{
		Capacity:       usage.Total,
		Remaining:      usage.Free,
		UsedPercent:    float32(usage.UsedPercent),
		LastUpdate:     time.Now().Unix(),
		MemUsedPercent: float32(v.UsedPercent),
	}

	return metrics, nil
}

func (n *Node) HandleGetBlock(block *transport_pb.BlockPlacementInfo) (*fs_pb.Block, error) {
	return n.data.ReadBlock(block)
}

func (n *Node) HandleAssignBlocks(blocks []*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {

	info, err := n.data.WriteBlocks(blocks)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (n *Node) Start() {

	go n.server.Run()
	go n.handleSignals()

	err := n.registerItself()
	if err != nil {
		logrus.Fatalf("can't register node in etcd: %v", err)
	}

	logrus.Infof("Node %v : %v is ready", n.Name, n.ID)

	select {
	case <-n.NotifyShutdown():
		return
	}
}

func (n *Node) registerItself() error {
	info := &transport_pb.NodeInfo{
		Id:       n.ID,
		Name:     n.Name,
		Address:  n.Config.Address,
		Location: n.Config.RhosusPath,
	}
	return n.etcd.RegisterNode(n.ID, info)
}

func (n *Node) Shutdown() error {
	logrus.Infof("shutting down node")

	n.data.Shutdown()

	err := n.unregisterItself()
	if err != nil {
		logrus.Errorf("error unregistering node: %v", err)
		return err
	}

	close(n.shutdownC)

	return err
}

func (n *Node) unregisterItself() error {
	return n.etcd.UnregisterNode(n.ID)
}

func (n *Node) NotifyShutdown() <-chan struct{} {
	return n.shutdownC
}

func (n *Node) handleSignals() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, os.Interrupt, syscall.SIGTERM)
	for {
		sig := <-sigc
		logrus.Infof("signal received: %v", sig)
		switch sig {
		case syscall.SIGHUP:

		case syscall.SIGINT, os.Interrupt, syscall.SIGTERM:
			pidFile := viper.GetString("pid_file")
			shutdownTimeout := time.Duration(viper.GetInt("shutdown_timeout")) * time.Second
			go time.AfterFunc(shutdownTimeout, func() {
				if pidFile != "" {
					os.Remove(pidFile)
				}
				os.Exit(1)
			})

			err := n.Shutdown()
			if err != nil {

			}

			if pidFile != "" {
				err := os.Remove(pidFile)
				if err != nil {
					logrus.Errorf("error removing pid file: %v", err)
				}
			}
			os.Exit(0)
		}
	}

}
