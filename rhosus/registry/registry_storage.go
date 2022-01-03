package registry

import (
	"fmt"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"sync"
)

const (
	defaultStorageFileName  = "indices.data"
	blocksStorageBucketName = "__blocks"
	metaStorageBucketName   = "__meta"
)

type RegistryStorage interface {
	Setup(existingNodes []string) error
	Close() error

	PutBlocks(nodeUid string, blocks map[string][]uint64) error
	RemoveBlocks(nodeUid string, blocks []string) error
}

type StorageConfig struct {
}

type Storage struct {
	Config StorageConfig

	db       *bolt.DB
	registry *Registry
	mu       sync.RWMutex
}

func NewStorage(config StorageConfig, registry *Registry) (*Storage, error) {

	db, err := bolt.Open(defaultStorageFileName, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &Storage{
		Config: config,

		registry: registry,
		db:       db,
	}, nil
}

func (s *Storage) Setup(existingNodes []string) error {

	err := s.db.Update(func(tx *bolt.Tx) error {

		b, err := tx.CreateBucketIfNotExists([]byte(blocksStorageBucketName))
		if err != nil {
			return err
		}

		for _, node := range existingNodes {
			_, err := b.CreateBucketIfNotExists([]byte(node))
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// <---------------------------------->
// Methods for working with data blocks
// <---------------------------------->

func (s *Storage) PutBlocks(nodeUid string, blocks map[string][]uint64) error {

	err := s.db.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blocksStorageBucketName))

		nodeB, err := b.CreateBucketIfNotExists([]byte(nodeUid))
		if err != nil {
			return err
		}

		for uid, rng := range blocks {

			// Normal range should only have 'from' and 'to' values
			// This actually can't happen, but we'd like to be sure

			if len(rng) != 2 {
				logrus.Errorf("invalid block range format %v", rng)
				continue
			}

			// For every block of data we create a bucket
			// I don't know yet what it takes on performance, will find out later

			rngBucket, err := nodeB.CreateBucketIfNotExists([]byte(uid))
			if err != nil {
				// todo something more thought out
				logrus.Errorf("error writing some blocks to storage: %v", err)
				continue
			}

			from := fmt.Sprintf("%v", rng[0])
			to := fmt.Sprintf("%v", rng[1])

			rngBucket.Put([]byte("at"), []byte(from))
			rngBucket.Put([]byte("to"), []byte(to))
		}

		return err
	})

	return err
}

func (s *Storage) RemoveBlocks(nodeUid string, blocks []string) error {

	err := s.db.Update(func(tx *bolt.Tx) error {

		b := tx.Bucket([]byte(blocksStorageBucketName))

		nodeB := b.Bucket([]byte(nodeUid))

		for _, uid := range blocks {
			err := nodeB.DeleteBucket([]byte(uid))
			if err != nil {
				// todo something more thought out
				continue
			}
		}

		return nil
	})

	return err
}
