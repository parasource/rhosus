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
	count := 0
	for _, part := range m.parts.parts {
		count += part.getUsedBlocks()
	}
	return count
}

func (m *Manager) GetPartitionsCount() int {
	return m.parts.getPartsCount()
}

func (m *Manager) WriteBlocks(blocks []*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return nil, ErrShutdown
	}

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
	//for _, part := range parts {
	//	go func(part *Partition) {
	//		defer wg.Done()
	//
	//		blocksCount := len(blocks) / len(parts)
	//		bSlice := blocks[offset : offset+blocksCount]
	//
	//		if true {
	//			for _, block := range bSlice {
	//				_ = part.sink.put(block)
	//			}
	//		} else {
	//			data := make(map[string][]byte)
	//			for _, block := range bSlice {
	//				data[block.Id] = block.Data
	//			}
	//
	//			err, errs := part.writeBlocks(data)
	//			if err != nil {
	//				logrus.Errorf("error writing to partition: %v", err)
	//				return
	//			}
	//			if len(errs) != 0 {
	//				// todo
	//			}
	//		}
	//
	//		for _, block := range bSlice {
	//			placement = append(placement, &transport_pb.BlockPlacementInfo{
	//				BlockID:     block.Id,
	//				PartitionID: part.ID,
	//				Success:     true,
	//			})
	//		}
	//	}(part)
	//}
	//wg.Wait()

	return placement, nil
}

func (m *Manager) ReadBlock(block *transport_pb.BlockPlacementInfo) (*fs_pb.Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return nil, ErrShutdown
	}

	part, err := m.parts.getPartition(block.PartitionID)
	if err != nil {
		return nil, err
	}

	b := part.readBlocks([]*transport_pb.BlockPlacementInfo{block})[block.BlockID]

	return b, nil
}
