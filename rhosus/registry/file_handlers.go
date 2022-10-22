/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package registry

import (
	"bytes"
	"crypto/md5"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (r *Registry) HandleGetFile(rw http.ResponseWriter, req *http.Request) error {
	filePath := strings.Trim(req.URL.Path, "/")

	rw.Header().Set("Accept-Ranges", "bytes")

	// Returns file

	err := r.GetFileHandler(filePath, func(block *fs_pb.Block) {
		reader := io.NopCloser(bytes.NewReader(block.Data))
		n, err := io.CopyN(rw, reader, int64(block.Len))
		if err != nil || n != int64(len(block.Data)) {
			logrus.Errorf("something went wrong: %v, %v - %v", err, n, len(block.Data))
		}
		//reader.Close()
	})
	if err != nil {
		switch err {
		case ErrNoSuchFileOrDirectory:
			rw.WriteHeader(404)
			return ErrNoSuchFileOrDirectory
		default:
			logrus.Errorf("error getting blocks: %v", err)
		}
	}

	return err
}

func (r *Registry) HandlePutFile(rw http.ResponseWriter, req *http.Request) error {
	var err error

	logrus.Infof(req.URL.String())

	multipartReader, err := req.MultipartReader()
	if err != nil {
		return err
	}

	part1, err := multipartReader.NextPart()
	if err != nil {
		return err
	}

	contentType := part1.Header.Get("Content-Type")
	if contentType == "application/octet-stream" {
		contentType = ""
	}

	md5Hash := md5.New()
	partReader := io.NopCloser(io.TeeReader(part1, md5Hash))

	var wg sync.WaitGroup

	var bytesBufferCounter int64
	bytesBufferLimitCond := sync.NewCond(new(sync.Mutex))
	//var fileChunksLock sync.Mutex

	blockSize := int64(2 << 20) // block size is 2mb

	counter := 1

	uid, _ := uuid.NewV4()
	filePath := strings.Trim(req.URL.String(), "/")
	filePathSplit := strings.Split(filePath, "/")
	fileName := filePathSplit[len(filePathSplit)-1]
	file := &control_pb.FileInfo{
		Id:          uid.String(),
		Name:        fileName,
		Type:        control_pb.FileInfo_FILE,
		Path:        filePath,
		Size_:       0,
		Permission:  nil,
		Owner:       "",
		Group:       "",
		Symlink:     "",
		Replication: 2,
	}
	err = r.RegisterFile(file)
	if err != nil {
		switch err {
		case ErrFileExists:
			rw.WriteHeader(http.StatusConflict)
			return nil
		case ErrNoSuchFileOrDirectory:
			rw.WriteHeader(http.StatusBadRequest)
			return nil
		default:
			logrus.Errorf("error registring file: %v", err)
			rw.WriteHeader(500)
			rw.Write([]byte("server error. see logs"))
			return nil
		}
	}

	var dataToTransfer []*fs_pb.Block
	for {
		bytesBufferLimitCond.L.Lock()
		for atomic.LoadInt64(&bytesBufferCounter) >= 4 {
			logrus.Infof("waiting for byte buffer %d", bytesBufferCounter)
			bytesBufferLimitCond.Wait()
		}
		atomic.AddInt64(&bytesBufferCounter, 1)
		bytesBufferLimitCond.L.Unlock()

		bytesBuffer := bufPool.Get().(*bytes.Buffer)

		limitedReader := io.LimitReader(partReader, blockSize)
		bytesBuffer.Reset()

		dataSize, err := bytesBuffer.ReadFrom(limitedReader)
		if err != nil || dataSize == 0 {
			bufPool.Put(bytesBuffer)
			atomic.AddInt64(&bytesBufferCounter, -1)
			bytesBufferLimitCond.Signal()
			break
		}

		var chunkOffset int64 = 0

		if dataSize < blockSize {
			func() {
				defer func() {
					bufPool.Put(bytesBuffer)
					atomic.AddInt64(&bytesBufferCounter, -1)
					bytesBufferLimitCond.Signal()
				}()
				smallContent := make([]byte, dataSize)
				bytesBuffer.Read(smallContent)

				dataReader := util.NewBytesReader(smallContent)

				// actual block payload
				data := dataReader.Bytes
				uid, _ = uuid.NewV4()
				block := &fs_pb.Block{
					Id:     uid.String(),
					Index:  uint64(counter),
					FileId: file.Id,
					Len:    uint64(len(data)),
					Data:   smallContent,
				}
				dataToTransfer = append(dataToTransfer, block)
			}()
			chunkOffset += dataSize

			break
		}

		wg.Add(1)
		go func(offset int64) {
			defer func() {
				bufPool.Put(bytesBuffer)
				atomic.AddInt64(&bytesBufferCounter, -1)
				bytesBufferLimitCond.Signal()
				wg.Done()
			}()

			dataReader := util.NewBytesReader(bytesBuffer.Bytes())

			// actual block payload
			data := dataReader.Bytes
			uid, _ = uuid.NewV4()
			block := &fs_pb.Block{
				Id:     uid.String(),
				Index:  uint64(counter),
				FileId: file.Id,
				Len:    uint64(len(data)),
				Data:   data,
			}
			dataToTransfer = append(dataToTransfer, block)

			counter++

		}(chunkOffset)

		chunkOffset = chunkOffset + dataSize

		// if last chunk was not at full chunk size, but already exhausted the reader
		if dataSize < blockSize {
			break
		}
	}

	err = r.TransportAndRegisterBlocks(file.Id, dataToTransfer, int(file.Replication))
	if err != nil {
		logrus.Errorf("error transporting blocks to node: %v", err)
	}

	rw.Write([]byte("OK"))

	return err
}

func (r *Registry) HandleDeleteFile(rw http.ResponseWriter, req *http.Request) error {
	return nil
}