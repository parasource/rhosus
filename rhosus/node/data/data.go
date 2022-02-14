package data

import (
	"github.com/parasource/rhosus/rhosus/backend"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"sync"
)

type Manager struct {
	parts   *PartitionsMap
	backend *backend.Storage

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
	//_, err = m.WriteBlocks(blocks)
	//logrus.Infof("wrote 4gb in %v", time.Since(start).String())

	return m, err
}

func (m *Manager) WriteBlocks(blocks map[string]*fs_pb.Block) ([]*transport_pb.BlockPlacementInfo, error) {
	m.mu.RLock()
	if m.shutdown {
		m.mu.RUnlock()
		return nil, ErrShutdown
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	data := make(map[string][]byte)
	for id, block := range blocks {
		data[id] = block.Data
	}

	var placement []*transport_pb.BlockPlacementInfo

	// todo: write to partitions in parallel
	for id, bytes := range data {
		p, err := m.parts.getRandomPartition()
		if err != nil {
			return nil, err
		}

		err, errs := p.writeBlocks(map[string][]byte{id: bytes})
		if err != nil {
			// todo
			continue
		}
		if len(errs) != 0 {
			// todo
		}

		placement = append(placement, &transport_pb.BlockPlacementInfo{
			BlockID:     id,
			PartitionID: p.ID,
		})
	}

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
