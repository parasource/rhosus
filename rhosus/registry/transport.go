/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package registry

import (
	"context"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/rs/zerolog/log"
	"sync"
)

type TransportConfig struct {
	WriteTimeoutMs int
}

type Transport struct {
	config TransportConfig

	mu    sync.RWMutex
	conns map[string]*transport_pb.TransportServiceClient

	shutdown  bool
	shutdownC chan struct{}
}

func NewTransport(config TransportConfig, conns map[string]*transport_pb.TransportServiceClient) *Transport {

	t := &Transport{
		config: config,
		conns:  make(map[string]*transport_pb.TransportServiceClient),

		shutdownC: make(chan struct{}),
	}

	for id, connR := range conns {
		conn := *connR
		if _, err := conn.Heartbeat(context.Background(), &transport_pb.HeartbeatRequest{}); err != nil {
			log.Error().Err(err).Msg("error adding conn to transport")
			continue
		}
		t.conns[id] = &conn
	}

	return t
}

func (t *Transport) AddConn(id string, conn *transport_pb.TransportServiceClient) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.shutdown {
		return
	}
	if _, ok := t.conns[id]; ok {
		return
	}

	t.conns[id] = conn
}

func (t *Transport) RemoveConn(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.shutdown {
		return
	}
	if _, ok := t.conns[id]; !ok {
		return
	}

	delete(t.conns, id)
}

func (t *Transport) Shutdown() {
	t.mu.Lock()
	t.shutdown = true
	t.mu.Unlock()

	close(t.shutdownC)
}

func (t *Transport) NotifyShutdown() <-chan struct{} {
	return t.shutdownC
}
