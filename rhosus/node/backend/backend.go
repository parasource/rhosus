package backend

import (
	"fmt"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"strconv"
	"sync"
)

const (
	defaultStorageFileName  = "indices.data"
	blocksStorageBucketName = "__blocks"
	metaStorageBucketName   = "__meta"
)

type BackendStorage interface {
	Setup(existingNodes []string) error
	Close() error

	PutBlocks(nodeUid string, blocks map[string][]uint64) error
	RemoveBlocks(nodeUid string, blocks []string) error
}

type StorageConfig struct {
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

		root := tx.Bucket([]byte(blocksStorageBucketName))

		for _, uid := range uids {
			b := root.Bucket([]byte(uid))

			block := Block{}
			block.From, _ = strconv.ParseUint(string(b.Get([]byte("from"))), 10, 64)
			block.To, _ = strconv.ParseUint(string(b.Get([]byte("to"))), 10, 64)
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

			// For every block of data we create a bucket
			// I don't know yet what it takes on performance, will find out later

			rngBucket, err := b.CreateBucketIfNotExists([]byte(uid))
			if err != nil {
				// todo something more thought out
				logrus.Errorf("error writing some blocks to storage: %v", err)
				continue
			}

			from := fmt.Sprintf("%v", rng.From)
			to := fmt.Sprintf("%v", rng.To)

			rngBucket.Put([]byte("from"), []byte(from))
			rngBucket.Put([]byte("to"), []byte(to))
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
