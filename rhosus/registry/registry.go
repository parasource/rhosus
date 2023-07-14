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
	"github.com/parasource/rhosus/rhosus/registry/storage"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/rs/zerolog/log"
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

	Storage *storage.Storage
	Cluster cluster.Config `json:"cluster"`
}

type Registry struct {
	Id     string
	Name   string
	mu     sync.RWMutex
	Config Config

	NodesMap *NodesMap
	Storage  *storage.Storage

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

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Address: config.EtcdAddr,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to etcd")
	}
	r.etcdClient = etcdClient

	r.Storage = config.Storage

	// Here we load all the existing nodes and registries from etcd
	// Error occurs only in non-usual conditions, so we shut down
	regs, err := r.getExistingRegistries()
	if err != nil {
		log.Fatal().Err(err).Msg("error getting existing registries from etcd")
	}

	nodes, err := r.getExistingNodes()
	if err != nil {
		log.Fatal().Err(err).Msg("error getting existing datanodes from etcd")
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
	r.Cluster.SetEntriesHandler(r.handleEntriesFromLeader)

	// Setting up nodes map from existing nodes
	nMap, err := NewNodesMap(r, nodes)
	if err != nil {

	}
	r.NodesMap = nMap

	// Registering itself in etcd cluster
	err = r.registerItself(info)
	if err != nil {
		log.Fatal().Err(err).Msg("error registering in etcd")
	}

	return r, nil
}

func (r *Registry) handleEntriesFromLeader(entries []*control_pb.Entry) {
	var err error
	for _, entry := range entries {
		switch entry.Type {
		case control_pb.Entry_ASSIGN_FILE:
			var entryAssign control_pb.EntryAssignFile
			err = entryAssign.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "assign_file").
					Msg("error unmarshalling entry")
				continue
			}
			err = r.registerFile(entryAssign.File)
			if err != nil {
				log.Error().Err(err).Str("file_id", entryAssign.File.Id).
					Msg("error registering file in storage")
			}
		case control_pb.Entry_DELETE_FILE:
			var entryDelete control_pb.EntryDeleteFile
			err = entryDelete.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "delete_file").
					Msg("error unmarshalling entry")
				continue
			}
			err = r.unregisterFile(entryDelete.File)
			if err != nil {
				log.Error().Err(err).Str("file_id", entryDelete.File.Id).
					Msg("error deleting file from storage")
			}
		case control_pb.Entry_ASSIGN_BLOCKS:
			var entryAssign control_pb.EntryAssignBlocks
			err = entryAssign.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "assign_blocks").
					Msg("error unmarshalling entry")
				continue
			}
			err = r.registerBlocks(entryAssign.Blocks)
			if err != nil {
				// todo more informative log
				log.Error().Err(err).
					Msg("error registering blocks in storage")
			}
		case control_pb.Entry_DELETE_BLOCKS:
			var entryDelete control_pb.EntryDeleteBlocks
			err = entryDelete.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "delete_blocks").Msg("error unmarshalling entry")
				continue
			}
			err = r.unregisterBlocks(entryDelete.Blocks)
			if err != nil {
				// todo more informative log
				log.Error().Err(err).
					Msg("error deleting blocks from storage")
			}
		case control_pb.Entry_CREATE_POLICY:
			var entryDelete control_pb.EntryCreatePolicy
			err = entryDelete.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "create_policy").Msg("error unmarshalling entry")
				continue
			}
			err = r.Storage.StorePolicy(entryDelete.Policy)
			if err != nil {
				// todo more informative log
				log.Error().Err(err).
					Msg("error creating policy")
			}
		case control_pb.Entry_DELETE_POLICY:
			var entryDelete control_pb.EntryDeletePolicy
			err = entryDelete.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "delete_policy").Msg("error unmarshalling entry")
				continue
			}
			policy, err := r.Storage.GetPolicy(entryDelete.PolicyName)
			if err != nil || policy == nil {
				log.Error().Err(err).Msg("error getting policy")
				continue
			}
			err = r.Storage.DeletePolicy(policy)
			if err != nil {
				// todo more informative log
				log.Error().Err(err).
					Msg("error deleting policy")
			}
		case control_pb.Entry_CREATE_TOKEN:
			var entryDelete control_pb.EntryCreateToken
			err = entryDelete.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "create_token").Msg("error unmarshalling entry")
				continue
			}
			err = r.Storage.StoreToken(entryDelete.Token)
			if err != nil {
				// todo more informative log
				log.Error().Err(err).
					Msg("error creating token")
			}
		case control_pb.Entry_REVOKE_TOKEN:
			var entryDelete control_pb.EntryRevokeToken
			err = entryDelete.Unmarshal(entry.Data)
			if err != nil {
				log.Error().Err(err).Str("type", "revoke_token").Msg("error unmarshalling entry")
				continue
			}
			token, err := r.Storage.GetToken(entryDelete.Accessor)
			if err != nil || token == nil {
				log.Error().Err(err).Msg("error getting token")
				continue
			}
			err = r.Storage.RevokeToken(token)
			if err != nil {
				// todo more informative log
				log.Error().Err(err).
					Msg("error creating token")
			}
		}
	}
}

///////////////////////////////////////////
// RegistriesMap instances management methods

func (r *Registry) Start() {
	go r.NodesMap.WatchNodes()
	go r.RunServiceDiscovery()

	r.readyC <- struct{}{}

	log.Info().Str("name", r.Name).Str("uuid", r.Id).Msg("registry is ready")

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

	log.Info().Msg("shutting down registry")
	pidFile := viper.GetString("pid_file")
	err := r.unregisterItself()
	if err != nil {
		log.Error().Err(err).Msg("error unregistering from etcd")
	}

	// todo add storage shutdown
	r.Cluster.Shutdown()

	if pidFile != "" {
		err := os.Remove(pidFile)
		if err != nil {
			log.Error().Err(err).Msg("error removing pid file")
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
		log.Fatal().Err(err).Msg("error getting existing registries")
	}

	result := make(map[string]*control_pb.RegistryInfo)

	for _, bytes := range registries {

		var info control_pb.RegistryInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling node info")
			continue
		}

		result[info.Id] = &info

		log.Info().Str("name", info.Name).Str("id", info.Id).Msg("registry added from existing map")
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
		log.Fatal().Err(err).Msg("error getting existing registries")
	}

	result := make(map[string]*transport_pb.NodeInfo)

	for _, bytes := range nodes {

		var info transport_pb.NodeInfo
		err := info.Unmarshal(bytes)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling node info")
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
				log.Error().Err(err).Msg("error pinging etcd")
			}

		case res := <-r.etcdClient.WatchForNodesUpdates():
			// Here we handle nodes updates

			for _, event := range res.Events {

				switch event.Type {

				// Node added or updated
				case clientv3.EventTypePut:
					data := string(event.Kv.Value)

					bytes, err := util.Base64Decode(data)
					if err != nil {
						log.Error().Err(err).Msg("error decoding base64 message from etcd")
					}

					var info transport_pb.NodeInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						log.Error().Err(err).Msg("error unmarshalling datanode info")
					}

					if r.NodesMap.NodeExists(info.Id) {
						// If node already exists in a node map => updating node info

						r.NodesMap.UpdateNodeInfo(info.Id, &info)
					} else {
						err := r.NodesMap.AddNode(info.Id, &info)
						if err != nil {
							log.Error().Err(err).Msg("error adding datanode")
							continue
						}

						log.Info().Str("name", info.Name).Str("id", info.Id).Msg("datanode added")
					}

				// Node shut down or errored
				case clientv3.EventTypeDelete:
					name := rhosus_etcd.ParseNodeName(string(event.Kv.Key))

					if r.NodesMap.NodeExists(name) {
						r.NodesMap.RemoveNode(name)

						log.Info().Str("name", name).Msg("datanode shut down")
					} else {
						log.Error().Str("name", name).Msg("undefined node deletion signal")
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
						log.Error().Err(err).Msg("error decoding base64 message from etcd")
					}

					var info control_pb.RegistryInfo
					err = info.Unmarshal(bytes)
					if err != nil {
						log.Error().Err(err).Msg("error unmarshalling registry info")
					}

					err = r.Cluster.DiscoverOrUpdate(info.Id, &info)
					if err != nil {
						log.Error().Err(err).Msg("error discovering registry")
					}

				case clientv3.EventTypeDelete:

					name := rhosus_etcd.ParseRegistryName(string(event.Kv.Key))

					if name == r.Name {
						continue
					}

					// todo handle registry shutdown

				}
			}
		}
	}
}
