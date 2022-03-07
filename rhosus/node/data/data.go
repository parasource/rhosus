package data

import (
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"sync"
)

type Manager struct {
	parts *PartitionsMap

	mu               sync.RWMutex
	shutdown         bool
	isReceivingPages bool
}

func NewManager() (*Manager, error) {

	m := &Manager{
		shutdown:         false,
		isReceivingPages: false,
	}

	pmap, err := NewPartitionsMap(defaultPartitionsDir, 1024)
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
			logrus.Errorf("error putting block in sink: %v", err)
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
		logrus.Errorf("error closing partitions map: %v", err)
	}
}
