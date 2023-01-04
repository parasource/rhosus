/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-memdb"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/rs/zerolog/log"
	"sort"
	"time"
)

var (
	ErrWriteTimeout = errors.New("backend storage write timeout")
	ErrShutdown     = errors.New("backend storage is shut down")
)

type Config struct {
	Path string

	WriteTimeoutS int
	NumWorkers    int
}

const (
	defaultFilesTableName  = "__files"
	defaultBlocksTableName = "__blocks"
)

type Storage struct {
	db      *memdb.MemDB
	backend StorageBackend

	flushIntervalS int
	flushBatchSize int
}

func NewStorage(conf Config) (*Storage, error) {
	db, err := setupMemoryStorage()
	if err != nil {
		return nil, fmt.Errorf("error setting up storage schemas: %w", err)
	}

	backend, err := NewFileStorageBackend(conf)
	if err != nil {
		return nil, fmt.Errorf("error creating storage backend: %w", err)
	}

	m := &Storage{
		db:      db,
		backend: backend,

		flushIntervalS: 5,
		flushBatchSize: 1000,
	}
	err = m.loadFromBackend()
	if err != nil {
		return nil, fmt.Errorf("error loading memory storage from backend: %w", err)
	}

	go func() {
		ticker := tickers.SetTicker(time.Second * time.Duration(m.flushIntervalS))
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				err := m.FlushToBackend()
				if err != nil {
					log.Error().Err(err).Msg("error saving memory storage to file")
				}
			}
		}
	}()

	return m, nil
}

func setupMemoryStorage() (*memdb.MemDB, error) {
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
					"parent_id": {
						Name:         "parent_id",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "ParentID"},
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
				},
			},
		},
	})
	return db, err
}

func (s *Storage) loadFromBackend() error {
	var err error

	files, err := s.backend.GetAllFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		err := s.StoreFile(file)
		if err != nil {
			log.Error().Err(err).Msg("error loading memory storage from file")
			continue
		}
	}

	blocks, err := s.backend.GetAllBlocks()
	if err != nil {
		return err
	}
	err = s.PutBlocks(blocks)
	if err != nil {
		return err
	}

	return err
}

func (s *Storage) StoreFile(file *control_pb.FileInfo) error {
	txn := s.db.Txn(true)

	err := txn.Insert(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}

	txn.Commit()

	return nil
}

func (s *Storage) GetFile(id string) (*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "id", id)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.FileInfo:
		return raw.(*control_pb.FileInfo), nil
	default:
		return nil, nil
	}
}

func (s *Storage) GetFileByPath(path string) (*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "path", path)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.FileInfo:
		return raw.(*control_pb.FileInfo), nil
	default:
		return nil, nil
	}
}

func (s *Storage) GetFilesByParentId(id string) ([]*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	var files []*control_pb.FileInfo
	res, err := txn.Get(defaultFilesTableName, "parent_id", id)
	if err != nil {
		return nil, err
	}

	for obj := res.Next(); obj != nil; obj = res.Next() {
		files = append(files, obj.(*control_pb.FileInfo))
	}

	return files, nil
}

func (s *Storage) PutBlocks(blocks []*control_pb.BlockInfo) error {
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

func (s *Storage) GetAllFiles() []*control_pb.FileInfo {
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

func (s *Storage) GetBlocks(fileID string) ([]*control_pb.BlockInfo, error) {
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

func (s *Storage) DeleteFileWithBlocks(file *control_pb.FileInfo) error {
	txn := s.db.Txn(true)

	res, err := txn.Get(defaultBlocksTableName, "file_id", file.Id)
	if err != nil {
		return err
	}
	for obj := res.Next(); obj != nil; obj = res.Next() {
		err := txn.Delete(defaultBlocksTableName, obj.(*control_pb.BlockInfo))
		if err != nil {
			txn.Abort()
			return err
		}
	}
	err = txn.Delete(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return nil
}

func (s *Storage) DeleteFile(file *control_pb.FileInfo) error {
	txn := s.db.Txn(true)
	err := txn.Delete(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return nil
}

func (s *Storage) DeleteBlocks(blocks []*control_pb.BlockInfo) error {
	txn := s.db.Txn(true)
	for _, block := range blocks {
		err := txn.Delete(defaultBlocksTableName, block)
		if err != nil {
			txn.Abort()
			return err
		}
	}
	txn.Commit()

	return nil
}

func (s *Storage) FlushToBackend() error {
	var err error

	txn := s.db.Txn(false)

	err = s.flushFilesToBackend(txn)
	if err != nil {
		return fmt.Errorf("error saving files to file: %w", err)
	}

	err = s.flushBlocksToBackend(txn)
	if err != nil {
		return fmt.Errorf("error flusing blocks batch to backend: %v", err)
	}

	return nil
}

func (s *Storage) flushFilesToBackend(txn *memdb.Txn) error {
	filesBatch := make(map[string]*control_pb.FileInfo, s.flushBatchSize)

	files, err := txn.Get(defaultFilesTableName, "id")
	if err != nil {
		return err
	}
	for obj := files.Next(); obj != nil; obj = files.Next() {
		if len(filesBatch) == s.flushBatchSize {
			err := s.backend.StoreFilesBatch(filesBatch)
			if err != nil {
				log.Error().Err(err).Msg("error flushing files batch to backend")
			}
			filesBatch = make(map[string]*control_pb.FileInfo, s.flushBatchSize)
		}
		file := obj.(*control_pb.FileInfo)
		filesBatch[file.Id] = file
	}
	err = s.backend.StoreFilesBatch(filesBatch)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) flushBlocksToBackend(txn *memdb.Txn) error {

	blocksBatch := make(map[string]*control_pb.BlockInfo)

	blocks, err := txn.Get(defaultBlocksTableName, "id")
	if err != nil {
		return err
	}
	for obj := blocks.Next(); obj != nil; obj = blocks.Next() {
		if len(blocksBatch) == s.flushBatchSize {
			err = s.backend.PutBlocksBatch(blocksBatch)
			if err != nil {
				// todo: probably retry later
			}
		}
		block := obj.(*control_pb.BlockInfo)
		blocksBatch[block.Id] = block
	}
	err = s.backend.PutBlocksBatch(blocksBatch)
	if err != nil {
		// todo: probably retry later
	}

	return nil
}
