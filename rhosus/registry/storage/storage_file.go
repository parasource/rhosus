package storage

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util"
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

var _ StorageBackend = &FileStorageBackend{}

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

		_, err = tx.CreateBucketIfNotExists([]byte(filesStorageBucketName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(blocksStorageBucketName))
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

//////////////////////////////
// ---------------------------
// file methods
// ---------------------------

func (s *FileStorageBackend) StoreFilesBatch(files map[string]*control_pb.FileInfo) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	batch := make(map[string]string, len(files))
	for filePath, file := range files {

		bytes, err := file.Marshal()
		if err != nil {
			return err
		}
		strBytes := util.Base64Encode(bytes)

		batch[filePath] = strBytes
	}

	// todo correct error handling
	err := s.db.Batch(func(tx *bolt.Tx) error {
		var err error

		b := tx.Bucket([]byte(filesStorageBucketName))
		for filePath, data := range batch {
			err = b.Put([]byte(filePath), []byte(data))
		}

		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *FileStorageBackend) GetFile(path string) (*control_pb.FileInfo, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var encodedFileBytes string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(filesStorageBucketName))
		encodedFileBytes = string(b.Get([]byte(path)))

		return nil
	})

	bytes, err := util.Base64Decode(encodedFileBytes)
	if err != nil {
		return nil, err
	}

	var file control_pb.FileInfo
	err = file.Unmarshal(bytes)
	if err != nil {
		log.Error().Err(err).Msg("error unmarshaling file info")
	}

	return &file, nil
}

func (s *FileStorageBackend) GetFilesBatch(paths []string) ([]*control_pb.FileInfo, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	// getting raw file infos, stored as base64 string
	var encodedFilesBytes []string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(filesStorageBucketName))
		for _, id := range paths {
			encodedFilesBytes = append(encodedFilesBytes, string(b.Get([]byte(id))))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var files []*control_pb.FileInfo
	for _, data := range encodedFilesBytes {
		bytes, err := util.Base64Decode(data)
		if err != nil {
			log.Error().Err(err).Msg("error decoding base64 file info")
			continue
		}

		var file control_pb.FileInfo
		err = file.Unmarshal(bytes)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshaling file info")
		}

		files = append(files, &file)
	}

	return files, nil
}

func (s *FileStorageBackend) GetAllFiles() ([]*control_pb.FileInfo, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var encodedFilesBytes []string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(filesStorageBucketName))

		err := b.ForEach(func(k, v []byte) error {
			encodedFilesBytes = append(encodedFilesBytes, string(v))

			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	var files []*control_pb.FileInfo
	for _, data := range encodedFilesBytes {
		bytes, err := util.Base64Decode(data)
		if err != nil {
			log.Error().Err(err).Msg("error decoding base64 file info")
			continue
		}

		var file control_pb.FileInfo
		err = file.Unmarshal(bytes)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshaling file info")
			continue
		}

		files = append(files, &file)
	}

	return files, nil
}

func (s *FileStorageBackend) DeleteFilesBatch(fileIDs []string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	// todo correct error handling
	err := s.db.Update(func(tx *bolt.Tx) error {
		var err error

		b := tx.Bucket([]byte(filesStorageBucketName))
		for _, id := range fileIDs {
			err = b.Delete([]byte(id))
		}

		return err
	})

	return err
}

/////////////////////////////
// --------------------------
// Blocks methods
// --------------------------

// PutBlocksBatch is used by node to map block ids to partitions
func (s *FileStorageBackend) PutBlocksBatch(blocks map[string]*control_pb.BlockInfo) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	data := make(map[string]string, len(blocks))
	for id, info := range blocks {
		bytes, err := info.Marshal()
		if err != nil {
			log.Error().Err(err).Msg("error marshaling block info")
			continue
		}
		data[id] = util.Base64Encode(bytes)
	}

	// todo correct error handling
	err := s.db.Batch(func(tx *bolt.Tx) error {
		var err error

		for blockID, partitionID := range data {
			b := tx.Bucket([]byte(blocksStorageBucketName))

			err = b.Put([]byte(blockID), []byte(partitionID))
		}

		return err
	})

	return err
}

func (s *FileStorageBackend) GetBlocksBatch(blocks []string) (map[string]string, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var res map[string]string

	err := s.db.View(func(tx *bolt.Tx) error {
		var err error

		b := tx.Bucket([]byte(blocksStorageBucketName))
		for _, id := range blocks {
			res[id] = string(b.Get([]byte(id)))
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *FileStorageBackend) GetAllBlocks() ([]*control_pb.BlockInfo, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	var blocksBytesEncoded []string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksStorageBucketName))

		err := b.ForEach(func(k, v []byte) error {
			blocksBytesEncoded = append(blocksBytesEncoded, string(v))

			return nil
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	var blocks []*control_pb.BlockInfo
	for _, data := range blocksBytesEncoded {
		bytes, err := util.Base64Decode(data)
		if err != nil {
			continue
		}

		var block control_pb.BlockInfo
		err = block.Unmarshal(bytes)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshaling block info")
			continue
		}

		blocks = append(blocks, &block)
	}

	return blocks, nil
}

func (s *FileStorageBackend) RemoveBlocksBatch(blockIDs []string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	// todo correct error handling
	err := s.db.Update(func(tx *bolt.Tx) error {
		var err error

		b := tx.Bucket([]byte(blocksStorageBucketName))
		for _, id := range blockIDs {
			err = b.Delete([]byte(id))
		}

		return err
	})

	return err
}
