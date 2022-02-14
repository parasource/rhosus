package registry

import (
	"github.com/hashicorp/go-memdb"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	defaultFilesTableName = "__files"
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
		},
	})
	if err != nil {
		return nil, err
	}

	return &MemoryStorage{
		registry: registry,
		db:       db,

		flushIntervalS: 5,
		flushBatchSize: 1000,
	}, nil
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

func (s *MemoryStorage) FlushToBackend() error {
	var err error

	txn := s.db.Txn(false)

	err = s.flushFilesToBackend(txn)
	if err != nil {
		logrus.Errorf("error flushing files batch to backend: %v", err)
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
