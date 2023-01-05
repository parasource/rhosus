package storage

import (
	"fmt"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"os"
	"path"
	"sync"
)

const (
	defaultDbFileName = "data.db"

	filesStorageBucketName  = "__files"
	blocksStorageBucketName = "__blocks"
	metaStorageBucketName   = "__meta"
)

var _ Backend = &FileStorageBackend{}

type FileStorageBackend struct {
	config Config
	db     *bolt.DB

	mu     sync.RWMutex
	closed bool
}

func NewFileStorageBackend(config Config) (*FileStorageBackend, error) {
	s := &FileStorageBackend{
		config: config,
		closed: false,
	}

	backendPath := config.Path
	err := os.Mkdir(backendPath, 0755)
	if os.IsExist(err) {
		// triggers if dir already exists
		log.Debug().Msg("backend path already exists, skipping")
	} else if err != nil {
		return nil, fmt.Errorf("error creating backend folder in %v: %v", config.Path, err)
	}

	db, err := bolt.Open(path.Join(backendPath, defaultDbFileName), 0666, nil)
	if err != nil {
		return nil, err
	}
	s.db = db

	err = s.setup()
	if err != nil {
		return nil, fmt.Errorf("error setting up backend: %v", err)
	}

	return s, nil
}

// setup creates necessary bboltdb buckets
func (s *FileStorageBackend) setup() (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		var err error

		_, err = tx.CreateBucketIfNotExists([]byte(EntryTypeFile))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(EntryTypeBlock))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(metaStorageBucketName))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *FileStorageBackend) Close() error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	return s.db.Close()
}

func (s *FileStorageBackend) Put(entry *Entry) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(entry.Type))
		return b.Put([]byte(entry.Key), entry.value)
	})
	if err != nil {
		return fmt.Errorf("error putting entries in storage: %w", err)
	}

	return nil
}

func (s *FileStorageBackend) Get(t EntryType, key string) (*Entry, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var entry Entry
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t))
		entry.value = b.Get([]byte(key))
		return nil
	})
	entry.Key = key
	entry.Type = t

	return &entry, nil
}

func (s *FileStorageBackend) Delete(t EntryType, key string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	return nil
}

func (s *FileStorageBackend) List(entryType EntryType) ([]*Entry, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var entries []*Entry
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(entryType))
		b.ForEach(func(k, v []byte) error {
			entry := &Entry{
				Type: entryType,
			}
			entry.Key = string(k)
			entry.value = v

			entries = append(entries, entry)
			return nil
		})
		return nil
	})

	return entries, nil
}
