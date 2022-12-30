/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package storage

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

func TestBackend_writeAndReadFiles(t *testing.T) {

	dbPath := path.Join(os.TempDir(), "test_files.db")

	b, err := NewStorage(Config{
		DbFilePath:    dbPath,
		WriteTimeoutS: 1,
		NumWorkers:    1,
	})
	assert.Nil(t, err)

	files := make(map[string]*control_pb.FileInfo, 1000)

	for i := 1; i <= 1000; i++ {

		fileId := fmt.Sprintf("index_%v.html", i)

		files[fileId] = &control_pb.FileInfo{
			Type:  control_pb.FileInfo_FILE,
			Id:    "123123",
			Path:  fmt.Sprintf("Desktop/index_%v.html", i),
			Size_: 64,
			Owner: "eovchinnikov",
			Group: "admin",
		}

		//for j := 0; j < 10; j++ {
		//	blocks[fileId] = append(blocks[fileId], &control_pb.BlockInfo{
		//		Index:  uint64(j),
		//		NodeID: "test_data_node",
		//		Size_:  10,
		//	})
		//}
	}

	defer os.Remove(dbPath)

	assert.Nil(t, b.StoreFilesBatch(files))

	for i := 1; i <= 1000; i += 99 {
		file, err := b.GetFile(fmt.Sprintf("index_%v.html", i))
		assert.Nil(t, err)

		assert.Equal(t, file.Id, files[fmt.Sprintf("index_%v.html", i)].Id)
	}

	var toDelete []string
	for id := range files {
		toDelete = append(toDelete, id)
	}
	assert.Nil(t, b.RemoveFilesBatch(toDelete))

	b.Shutdown()
}

func TestBackend_writeBlocks(t *testing.T) {

	dbPath := "./test_blocks.db"

	b, err := NewStorage(Config{
		DbFilePath:    dbPath,
		WriteTimeoutS: 1,
		NumWorkers:    1,
	})
	assert.Nil(t, err)

	blocks := make(map[string]string, 10000)

	for i := 1; i <= 1000; i++ {
		blocks[fmt.Sprintf("block_%v", i)] = fmt.Sprintf("partition_%v", i)
	}
	defer os.Remove(dbPath)

	//assert.Nil(t, b.PutBlocksBatch(blocks))

	var toDelete []string
	for id := range blocks {
		toDelete = append(toDelete, id)
	}
	assert.Nil(t, b.RemoveBlocksBatch(toDelete))

	b.Shutdown()
}
