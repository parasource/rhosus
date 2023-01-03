/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package registry

import (
	"fmt"
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/registry/cluster"
	"github.com/parasource/rhosus/rhosus/storage"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
	"sync"
	"time"
)

const (
	etcdPingInterval = 1
)

// Config is a main configuration of Registry node
type Config struct {
	ID          string // it's being set dynamically
	ApiAddr     string `json:"api_addr"`
	EtcdAddr    string `json:"etcd_addr"`
	StoragePath string `json:"storage_path"`
	RhosusPath  string `json:"rhosus_path"`

	Backend storage.Config `json:"backend"`
	Cluster cluster.Config `json:"cluster"`
}

type Registry struct {
	Id     string
	Name   string
	mu     sync.RWMutex
	Config Config

	NodesManager   *NodesMap
	Backend        *storage.Storage
	MemoryStorage  *MemoryStorage
	StatsCollector *StatsCollector

	// Cluster is used to control over other registries
	Cluster *cluster.Cluster

	etcdClient *rhosus_etcd.EtcdClient

	readyC  chan struct{}
	readyWg sync.WaitGroup

	shutdown  bool
	shutdownC chan struct{}
}

func NewRegistry(config Config) (*Registry, error) {
	r := &Registry{
		Id:      config.ID,
		Name:    util.GenerateRandomName(2),
		Config:  config,
		readyWg: sync.WaitGroup{},

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),

		shutdown: false,
	}

	statsCollector := NewStatsCollector(r, 5)
	r.StatsCollector = statsCollector

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Address: config.EtcdAddr,
	})
	if err != nil {
		logrus.Fatalf("error connecting to etcd: %v", err)
	}
	r.etcdClient = etcdClient

	s, err := storage.NewStorage(config.Backend)
	if err != nil {
		logrus.Fatalf("error creating storage: %v", err)
	}
	r.Backend = s

	memStorage, err := NewMemoryStorage(r)
	if err != nil {
		logrus.Fatalf("error creating memory storage: %v", err)
	}
	r.MemoryStorage = memStorage

	// Here we load all the existing nodes and registries from etcd
	// Error occurs only in non-usual conditions, so we shut down
	regs, err := r.getExistingRegistries()
	if err != nil {
		logrus.Fatalf("error getting existing registries from etcd: %v", err)
	}

	nodes, err := r.getExistingNodes()
	if err != nil {
		logrus.Fatalf("error getting existing nodes from etcd: %v", err)
	}

	info := &control_pb.RegistryInfo{
		Id:      r.Id,
		Name:    r.Name,
		Address: config.Cluster.ClusterAddr,
	}

	// Setting up registries cluster from existing peers
	c, err := cluster.NewCluster(config.Cluster, regs)
	if err != nil {
		return nil, fmt.Errorf("error setting up cluster: %w", err)
	}
	r.Cluster = c
	r.Cluster.SetRegistryInfo(info)

	// Setting up nodes map from existing nodes
	nMap, err := NewNodesMap(r, nodes)
	if err != nil {

	}
	r.NodesManager = nMap

	// Registering itself in etcd cluster
	err = r.registerItself(info)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	return r, nil
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) Start() {

	go r.NodesManager.WatchNodes()
	go r.RunServiceDiscovery()

	go r.StatsCollector.Run()

	r.readyC <- struct{}{}

	logrus.Infof("Registry %v:%v is ready", r.Name, r.Id)

	select {
	case <-r.NotifyShutdown():
		return
	}

}

func (r *Registry) Shutdown() {
	r.mu.RLock()
	if r.shutdown {
		r.mu.RUnlock()
		return
	}
	r.mu.RUnlock()

	close(r.shutdownC)

	logrus.Infof("shutting down registry")
	pidFile := viper.GetString("pid_file")
	err := r.unregisterItself()
	if err != nil {
		logrus.Errorf("error unregistering: %v", err)
	}

	r.Backend.Shutdown()
	r.Cluster.Shutdown()

	if pidFile != "" {
		err := os.Remove(pidFile)
		if err != nil {
			logrus.Errorf("error removing pid file: %v", err)
		}
	}

	os.Exit(0)
}

func (r *Registry) registerItself(info *control_pb.RegistryInfo) error {
	r.mu.RLock()
	if r.shutdown {
		r.mu.RUnlock()
		return nil
	}
	r.mu.RUnlock()

	return r.etcdClient.RegisterRegistry(info.Id, info)
}

func (r *Registry) unregisterItself() error {
	r.mu.RLock()
	if r.shutdown {
		r.mu.RUnlock()
		return nil
	}
	r.mu.RUnlock()

	return r.etcdClient.UnregisterRegistry(r.Id)
}

func (r *Registry) NotifyShutdown() <-chan struct{} {
	return r.shutdownC
}

func (r *Registry) NotifyReady() <-chan struct{} {
	return r.readyC
}

func (r *Registry) getExistingRegistries() (map[string]*control_pb.RegistryInfo, error) {
	r.mu.RLock()
	if r.shutdown {
		r.mu.RUnlock()
		return map[string]*control_pb.RegistryInfo{}, nil
	}
	r.mu.RUnlock()

	registries, err := r.etcdClient.GetExistingRegistries()
	if err != nil {
		logrus.Fatalf("error getting existing registries: %v", err)
	}

	result := make(map[string]*control_pb.RegistryInfo)

	for _, bytes := range registries {

		var info control_pb.RegistryInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling node info: %v", err)
			continue
		}

		result[info.Id] = &info

		logrus.Infof("registry %v added from existing map", info.Name)
	}

	return result, nil
}

func (r *Registry) getExistingNodes() (map[string]*transport_pb.NodeInfo, error) {
	r.mu.RLock()
	if r.shutdown {
		r.mu.RUnlock()
		return map[string]*transport_pb.NodeInfo{}, nil
	}
	r.mu.RUnlock()

	nodes, err := r.etcdClient.GetExistingNodes()
	if err != nil {
		logrus.Fatalf("error getting existing registries: %v", err)
	}

	result := make(map[string]*transport_pb.NodeInfo)

	for _, bytes := range nodes {

		var info transport_pb.NodeInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling node info: %v", err)
			continue
		}

		result[info.Id] = &info
	}

	return result, nil
}

// <------------------------------------->
// Nodes and registries management methods
// <------------------------------------->

func (r *Registry) RunServiceDiscovery() {
	r.mu.RLock()
	if r.shutdown {
		r.mu.RUnlock()
		return
	}
	r.mu.RUnlock()

	// Waiting for new nodes to connect

	ticker := tickers.SetTicker(time.Second * time.Duration(etcdPingInterval))
	defer ticker.Stop()

	for {

		select {
		case <-ticker.C:

			err := r.etcdClient.Ping()
			if err != nil {
				logrus.Errorf("error pinging etcd: %v", err)
			}

		case res := <-r.etcdClient.WatchForNodesUpdates():
			// Here we handle nodes updates

			for _, event := range res.Events {

				switch event.Type {

				// Node added or updated
				case clientv3.EventTypePut:

					logrus.Info("RECEIVED EVENT PUT")

					data := string(event.Kv.Value)

					bytes, err := util.Base64Decode(data)
					if err != nil {
						logrus.Errorf("error decoding")
					}

					var info transport_pb.NodeInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						logrus.Errorf("error unmarshaling node info: %v", err)
					}

					if r.NodesManager.NodeExists(info.Id) {
						// If node already exists in a node map => updating node info

						r.NodesManager.UpdateNodeInfo(info.Id, &info)
					} else {
						err := r.NodesManager.AddNode(info.Id, &info)
						if err != nil {
							logrus.Errorf("error adding node: %v", err)
							continue
						}

						logrus.Infof("node %v added", info.Name)
					}

				// Node shut down or errored
				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseNodeName(string(event.Kv.Key))

					if r.NodesManager.NodeExists(name) {
						r.NodesManager.RemoveNode(name)

						logrus.Infof("node %v shut down", name)
					} else {
						logrus.Warnf("undefined node deletion signal")
					}

				}
			}

		case res := <-r.etcdClient.WatchForRegistriesUpdates():
			// Here we handle registries updates

			for _, event := range res.Events {

				switch event.Type {

				case clientv3.EventTypePut:

					name := string(event.Kv.Key)
					data := string(event.Kv.Value)

					if name == r.Name {
						continue
					}

					bytes, err := util.Base64Decode(data)
					if err != nil {
						logrus.Errorf("error decoding")
					}

					var info control_pb.RegistryInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						logrus.Errorf("error unmarshaling registry info: %v", err)
					}

					err = r.Cluster.DiscoverOrUpdate(info.Id, &info)
					if err != nil {
						logrus.Errorf("error discovering registry: %v", err)
					}
					//if r.RegistriesMap.RegistryExists(name) {
					//
					//} else {
					//	err := r.RegistriesMap.Add(name, &info)
					//	if err != nil {
					//		logrus.Errorf("error adding new registry: %v", err)
					//	}
					//}

				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseRegistryName(string(event.Kv.Key))

					if name == r.Name {
						continue
					}

					//if r.RegistriesMap.RegistryExists(name) {
					//	err := r.RegistriesMap.Remove(name)
					//	if err != nil {
					//		logrus.Errorf("error removing registry: %v", err)
					//	}
					//} else {
					//	logrus.Warn("undefined registry deletion signal")
					//}

				}
			}
		}
	}
}
