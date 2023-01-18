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
	"github.com/rs/zerolog/log"
	"sync"
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

type Storage struct {
	mu      sync.RWMutex
	db      *memdb.MemDB
	backend Backend

	flushIntervalS int
	flushBatchSize int

	closed bool
}

func NewStorage(conf Config) (*Storage, error) {
	db, err := setupMemDB()
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

	return m, nil
}

func (s *Storage) SetBackend(backend Backend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	s.backend = backend
}

func setupMemDB() (*memdb.MemDB, error) {
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

			defaultRolesTableName: {
				Name: defaultRolesTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "ID"},
					},
					"name": {
						Name:         "name",
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Name"},
					},
				},
			},

			defaultTokensTableName: {
				Name: defaultTokensTableName,
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:         "id",
						AllowMissing: false,
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Token"},
					},
					"role_id": {
						Name:         "role_id",
						AllowMissing: false,
						Unique:       false,
						Indexer:      &memdb.StringFieldIndex{Field: "RoleID"},
					},
				},
			},
		},
	})
	return db, err
}

func (s *Storage) loadFromBackend() error {
	var err error

	fileEntries, err := s.backend.List(EntryTypeFile)
	if err != nil {
		return err
	}
	for _, entry := range fileEntries {
		var file control_pb.FileInfo

		err := file.Unmarshal(entry.Value)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling file info")
			continue
		}

		err = s.storeFileInMemory(&file)
		if err != nil {
			log.Error().Err(err).Msg("error loading file from backend")
			continue
		}
	}

	blockEntries, err := s.backend.List(EntryTypeBlock)
	if err != nil {
		return err
	}
	for _, entry := range blockEntries {
		var block control_pb.BlockInfo

		err := block.Unmarshal(entry.Value)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling block info")
			continue
		}

		err = s.storeBlockInMemory(&block)
		if err != nil {
			log.Error().Err(err).Msg("error loading block from backend")
			continue
		}
	}

	roleEntries, err := s.backend.List(EntryTypeRole)
	if err != nil {
		return err
	}
	for _, entry := range roleEntries {
		var role control_pb.Role

		err := role.Unmarshal(entry.Value)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling role info")
			continue
		}

		err = s.storeRoleInMemory(&role)
		if err != nil {
			log.Error().Err(err).Msg("error loading role from backend")
			continue
		}
	}

	tokenEntries, err := s.backend.List(EntryTypeToken)
	if err != nil {
		return err
	}
	for _, entry := range tokenEntries {
		var token control_pb.Token

		err := token.Unmarshal(entry.Value)
		if err != nil {
			log.Error().Err(err).Msg("error unmarshalling token info")
			continue
		}

		err = s.storeTokenInMemory(&token)
		if err != nil {
			log.Error().Err(err).Msg("error loading token from backend")
			continue
		}
	}

	return err
}
