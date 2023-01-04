/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package data

import (
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/rs/zerolog/log"
	"sync"
)

type Manager struct {
	parts *PartitionsMap

	mu               sync.RWMutex
	shutdown         bool
	isReceivingPages bool
}

func NewManager(dir string) (*Manager, error) {
	m := &Manager{
		shutdown:         false,
		isReceivingPages: false,
	}

	pmap, err := NewPartitionsMap(dir, 1024)
	if err != nil {
		return nil, err
	}
	m.parts = pmap

	return m, err
}

func (m *Manager) GetBlocksCount() int {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return 0
	}
	m.mu.RUnlock()

	count := 0
	for _, part := range m.parts.parts {
		count += part.GetUsedBlocks()
	}
	return count
}

func (m *Manager) GetPartitionsCount() int {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return 0
	}
	m.mu.RUnlock()

	return m.parts.getPartsCount()
}

func (m *Manager) WriteBlocks(blocks []*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return nil, ErrShutdown
	}
	m.mu.RUnlock()

	//var wg sync.WaitGroup
	var placement []*transport_pb.BlockPlacementInfo

	parts := m.parts.GetNotFullPartitions()

	// just in case new part wasn't created by watcher
	if len(parts) == 0 {
		newPartId, err := m.parts.createPartition()
		if err != nil {
			return nil, err
		}

		part, _ := m.parts.getPartition(newPartId)
		parts = append(parts, part)
	}

	for i, block := range blocks {
		partIdx := i % len(parts)

		part := parts[partIdx]

		err := part.sink.put(block)
		if err != nil {
			log.Error().Err(err).Msg("error putting block in sink")
			continue
		}

		placement = append(placement, &transport_pb.BlockPlacementInfo{
			BlockID:     block.Id,
			PartitionID: parts[partIdx].ID,
			Success:     true,
		})
	}

	return placement, nil
}

func (m *Manager) RemoveBlocks(blocks []*transport_pb.BlockPlacementInfo) error {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return ErrShutdown
	}
	m.mu.RUnlock()

	for _, block := range blocks {
		part, err := m.parts.getPartition(block.PartitionID)
		if err != nil {
			return err
		}

		err = part.RemoveBlocks([]*transport_pb.BlockPlacementInfo{block})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) ReadBlock(block *transport_pb.BlockPlacementInfo) (*fs_pb.Block, error) {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return nil, ErrShutdown
	}
	m.mu.RUnlock()

	part, err := m.parts.getPartition(block.PartitionID)
	if err != nil {
		return nil, err
	}

	bs, err := part.ReadBlocks([]*transport_pb.BlockPlacementInfo{block})
	if err != nil {
		return nil, err
	}

	return bs[block.BlockID], nil
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	if m.shutdown {
		m.mu.Unlock()
		return
	}
	m.shutdown = true
	m.mu.Unlock()

	err := m.parts.Shutdown()
	if err != nil {
		log.Error().Err(err).Msg("error closing partitions map")
	}
}
