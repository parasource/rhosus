package data

import (
	"github.com/parasource/rhosus/rhosus/backend"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"sync"
)

type Manager struct {
	parts   *PartitionsMap
	backend *backend.Storage // for now it is not useful, but i guess i will use it later

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

	b, err := backend.NewStorage(backend.Config{
		DbFilePath:    "indices.db",
		WriteTimeoutS: 1,
		NumWorkers:    1,
	})
	if err != nil {
		return nil, err
	}
	m.backend = b

	return m, err
}

func (m *Manager) WriteBlocks(blocks []*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return nil, ErrShutdown
	}

	var wg sync.WaitGroup
	var placement []*transport_pb.BlockPlacementInfo

	parts := m.parts.GetAvailablePartitions(len(blocks))
	if len(parts) == 0 {
		newPartId, err := m.parts.createPartition()
		if err != nil {
			return nil, err
		}

		parts[newPartId], _ = m.parts.getPartition(newPartId)
	}

	offset := 0
	for _, part := range parts {
		wg.Add(1)
		go func(part *Partition, offset int) {
			defer wg.Done()

			blocksCount := len(blocks) / len(parts)
			bSlice := blocks[offset : offset+blocksCount]

			data := make(map[string][]byte)
			for _, block := range bSlice {
				data[block.Id] = block.Data
			}

			err, errs := part.writeBlocks(data)
			if err != nil {
				logrus.Errorf("error writing to partition: %v", err)
				return
			}
			if len(errs) != 0 {
				// todo
			}

			for _, block := range bSlice {
				placement = append(placement, &transport_pb.BlockPlacementInfo{
					BlockID:     block.Id,
					PartitionID: part.ID,
					Success:     true,
				})
			}
		}(part, offset)
	}

	wg.Wait()

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

	// todo: REFACTOR!!
	b := &fs_pb.Block{
		Id: block.BlockID,
	}
	b.Data = part.readBlocks([]*transport_pb.BlockPlacementInfo{block})[block.BlockID]

	return b, nil
}
