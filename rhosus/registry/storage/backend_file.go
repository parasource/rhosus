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

	return s, nil
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

func (s *FileStorageBackend) Put(t EntryType, entries []*Entry) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	err := s.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(t))
		if err != nil {
			return err
		}

		for _, entry := range entries {
			err := b.Put([]byte(entry.Key), entry.value)
			if err != nil {
				// todo correct error handling
				log.Error().Err(err).Msg("error putting entry in backend")
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error putting entries in storage: %w", err)
	}

	return nil
}

func (s *FileStorageBackend) Get(t EntryType, keys []string) ([]*Entry, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var entries []*Entry
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t))
		for _, key := range keys {
			val := b.Get([]byte(key))
			entries = append(entries, &Entry{
				Key:   key,
				value: val,
			})
		}
		return nil
	})

	return entries, nil
}

func (s *FileStorageBackend) Delete(t EntryType, keys []string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t))
		for _, key := range keys {
			err := b.Delete([]byte(key))
			if err != nil {
				// todo correct error handling
				log.Error().Err(err).Msg("error deleting entry from backend")
			}
		}
		return nil
	})

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
	err := s.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(entryType))
		if err != nil {
			return err
		}
		b.ForEach(func(k, v []byte) error {
			entry := &Entry{
				Key:   string(k),
				value: v,
			}

			entries = append(entries, entry)
			return nil
		})
		return nil
	})

	return entries, err
}
