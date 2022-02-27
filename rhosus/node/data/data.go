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

	//time.Sleep(time.Second)
	//
	//var data []byte
	//for i := 0; i < 15*1024*1024; i++ {
	//	data = append(data, byte('a'))
	//}
	//blocks := make(map[string]*fs_pb.Block, 256)
	//start := time.Now()
	//for i := 0; i < 256; i++ {
	//	uid, _ := uuid.NewV4()
	//
	//	blocks[uid.String()] = &fs_pb.Block{
	//		Id:      uid.String(),
	//		FileId:  "123123",
	//		Size_:   64,
	//		Data:    data,
	//	}
	//}
	//res, err := m.WriteBlocks(blocks)
	//logrus.Info(res)
	//logrus.Infof("wrote 4gb in %v", time.Since(start).String())

	return m, err
}

func (m *Manager) WriteBlocks(blocks []*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return nil, ErrShutdown
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *Manager) ReadBlocks(IDs string) (map[string]*fs_pb.Block, error) {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return nil, ErrShutdown
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	return nil, nil
}
