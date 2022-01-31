package backend

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"strconv"
	"sync"
)

const (
	defaultStorageFileName  = "indices.data"
	blocksStorageBucketName = "__blocks"
	metaStorageBucketName   = "__meta"
)

type StorageConfig struct {
	StorageFileName string
}

type Storage struct {
	Config StorageConfig

	backend *bolt.DB
	mu      sync.RWMutex
	data    map[string][]byte
}

func NewBackend(config StorageConfig) (*Storage, error) {

	db, err := bolt.Open(defaultStorageFileName, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &Storage{
		Config: config,

		backend: db,
	}, nil
}

func (s *Storage) Close() error {
	return s.backend.Close()
}

type Block struct {
	From uint64
	To   uint64
	Size int64
}

func (s *Storage) GetBlocks(uids []string) (map[string]Block, error) {
	blocks := make(map[string]Block, len(uids))
	err := s.backend.Batch(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blocksStorageBucketName))

		for _, uid := range uids {

			block := Block{}
			block.From, _ = strconv.ParseUint(string(b.Get([]byte(blockFromKey(uid)))), 10, 64)
			block.To, _ = strconv.ParseUint(string(b.Get([]byte(blockToKey(uid)))), 10, 64)
			blocks[uid] = block

		}

		return nil
	})

	return blocks, err
}

func (s *Storage) PutBlocks(blocks map[string]Block) error {

	err := s.backend.Update(func(tx *bolt.Tx) error {

		b, _ := tx.CreateBucketIfNotExists([]byte(blocksStorageBucketName))

		for uid, rng := range blocks {

			from := fmt.Sprintf("%v", rng.From)
			to := fmt.Sprintf("%v", rng.To)

			b.Put([]byte(blockFromKey(uid)), []byte(from))
			b.Put([]byte(blockToKey(uid)), []byte(to))
		}

		return nil
	})

	return err
}

func (s *Storage) RemoveBlocks(blocks []string) error {

	err := s.backend.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blocksStorageBucketName))

		for _, uid := range blocks {
			err := b.DeleteBucket([]byte(uid))
			if err != nil {
				// todo something more thought out
				continue
			}
		}

		return nil
	})

	return err
}
