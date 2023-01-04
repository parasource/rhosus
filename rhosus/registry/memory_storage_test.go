/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package registry

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/storage"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

var (
	dbPath = path.Join(os.TempDir(), "test.db")
)

const (
	filesCount = 10000
)

func mockRegistry(t *testing.T) *Registry {
	t.Helper()

	b, err := storage.NewStorage(storage.Config{
		DbFilePath:    dbPath,
		WriteTimeoutS: 1,
		NumWorkers:    1,
	})
	assert.Nil(t, err)

	return &Registry{
		Backend: b,
	}
}

func TestMemoryStorage_FlushFiles(t *testing.T) {
	r := mockRegistry(t)
	s, err := NewMemoryStorage(r)
	assert.Nil(t, err)

	for i := 1; i <= filesCount; i++ {
		err := s.StoreFile(&control_pb.FileInfo{
			Type:  control_pb.FileInfo_FILE,
			Id:    fmt.Sprintf("index_%v.html", i),
			Path:  fmt.Sprintf("Desktop/index_%v.html", i),
			Size_: 64,
			Owner: "eovchinnikov",
			Group: "admin",
		})
		assert.Nil(t, err)
	}

	txn := s.db.Txn(false)
	err = s.flushFilesToBackend(txn)
	assert.Nil(t, err)

	r.Backend.Shutdown()
	os.Remove(dbPath)
}
