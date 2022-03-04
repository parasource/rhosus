package registry

import (
	"github.com/hashicorp/go-memdb"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"sort"
	"time"
)

const (
	defaultFilesTableName  = "__files"
	defaultBlocksTableName = "__blocks"
)

type MemoryStorage struct {
	registry *Registry

	db *memdb.MemDB

	flushIntervalS int
	flushBatchSize int
}

func NewMemoryStorage(registry *Registry) (*MemoryStorage, error) {

	db, err := memdb.NewMemDB(&memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{

			// Files table schema
			defaultFilesTableName: {
				Name: defaultFilesTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Id"},
					},
					"path": {
						Name:         "path",
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Path"},
					},
				},
			},

			// Blocks table schema
			defaultBlocksTableName: {
				Name: defaultBlocksTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Id"},
					},
					"file_id": {
						Name:         "file_id",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "FileID"},
					},
					"partition_id": {
						Name:         "partition_id",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "PartitionID"},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	m := &MemoryStorage{
		registry: registry,
		db:       db,

		flushIntervalS: 5,
		flushBatchSize: 1000,
	}
	err = m.loadFromBackend()
	if err != nil {
		logrus.Errorf("error loading memory storage from backend: %v", err)
		return nil, err
	}

	return m, nil
}

func (s *MemoryStorage) Start() {
	ticker := tickers.SetTicker(time.Second * time.Duration(s.flushIntervalS))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := s.FlushToBackend()
			if err != nil {
				logrus.Errorf("error flushing memory to backend: %v", err)
			}
		case <-s.registry.NotifyShutdown():
			return
		}
	}
}

func (s *MemoryStorage) loadFromBackend() error {
	var err error

	files, err := s.registry.Backend.GetAllFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		err := s.StoreFile(file)
		if err != nil {
			logrus.Errorf("error loading file from backend to mem storage: %v", err)
			continue
		}
	}

	blocks, err := s.registry.Backend.GetAllBlocks()
	if err != nil {
		return err
	}
	err = s.PutBlocks(blocks)
	if err != nil {
		return err
	}

	return err
}

func (s *MemoryStorage) StoreFile(file *control_pb.FileInfo) error {

	txn := s.db.Txn(true)

	err := txn.Insert(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}

	txn.Commit()

	return nil
}

func (s *MemoryStorage) GetFile(id string) (*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "id", id)
	if err != nil {
		return nil, err
	}

	return raw.(*control_pb.FileInfo), nil
}

func (s *MemoryStorage) GetFileByPath(path string) (*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "path", path)
	if err != nil {
		return nil, err
	}

	return raw.(*control_pb.FileInfo), nil
}

func (s *MemoryStorage) PutBlocks(blocks []*control_pb.BlockInfo) error {
	var err error

	txn := s.db.Txn(true)
	for _, block := range blocks {
		err = txn.Insert(defaultBlocksTableName, block)
		if err != nil {
			txn.Abort()
			return err
		}
	}
	txn.Commit()

	return nil
}

func (s *MemoryStorage) GetAllFiles() []*control_pb.FileInfo {
	txn := s.db.Txn(false)

	var files []*control_pb.FileInfo
	res, err := txn.Get(defaultFilesTableName, "id")
	if err != nil {
		return nil
	}

	for obj := res.Next(); obj != nil; obj = res.Next() {
		files = append(files, obj.(*control_pb.FileInfo))
	}

	return files
}

func (s *MemoryStorage) GetBlocks(fileID string) ([]*control_pb.BlockInfo, error) {
	txn := s.db.Txn(false)

	var blocks []*control_pb.BlockInfo

	res, err := txn.Get(defaultBlocksTableName, "file_id", fileID)
	if err != nil {
		return nil, err
	}
	for obj := res.Next(); obj != nil; obj = res.Next() {
		blocks = append(blocks, obj.(*control_pb.BlockInfo))
	}

	// sort blocks in sequence order
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Index < blocks[j].Index
	})

	return blocks, nil
}

func (s *MemoryStorage) FlushToBackend() error {
	var err error

	txn := s.db.Txn(false)

	err = s.flushFilesToBackend(txn)
	if err != nil {
		logrus.Errorf("error flushing files batch to backend: %v", err)
	}

	err = s.flushBlocksToBackend(txn)
	if err != nil {
		logrus.Errorf("error flusing blocks batch to backend: %v", err)
	}

	return err
}

func (s *MemoryStorage) flushFilesToBackend(txn *memdb.Txn) error {
	filesBatch := make(map[string]*control_pb.FileInfo, s.flushBatchSize)

	files, err := txn.Get(defaultFilesTableName, "id")
	if err != nil {
		return err
	}
	for obj := files.Next(); obj != nil; obj = files.Next() {
		if len(filesBatch) == s.flushBatchSize {
			err := s.registry.Backend.StoreFilesBatch(filesBatch)
			if err != nil {
				logrus.Errorf("error flushing files batch to backend: %v", err)
			}
			filesBatch = make(map[string]*control_pb.FileInfo, s.flushBatchSize)
		}
		file := obj.(*control_pb.FileInfo)
		filesBatch[file.Id] = file
	}
	err = s.registry.Backend.StoreFilesBatch(filesBatch)
	if err != nil {
		return err
	}

	return nil
}

func (s *MemoryStorage) flushBlocksToBackend(txn *memdb.Txn) error {

	blocksBatch := make(map[string]*control_pb.BlockInfo)

	blocks, err := txn.Get(defaultBlocksTableName, "id")
	if err != nil {
		return err
	}
	for obj := blocks.Next(); obj != nil; obj = blocks.Next() {
		if len(blocksBatch) == s.flushBatchSize {
			err = s.registry.Backend.PutBlocksBatch(blocksBatch)
			if err != nil {
				// todo: probably retry later
			}
		}
		block := obj.(*control_pb.BlockInfo)
		blocksBatch[block.Id] = block
	}
	err = s.registry.Backend.PutBlocksBatch(blocksBatch)
	if err != nil {
		// todo: probably retry later
	}

	return nil
}
