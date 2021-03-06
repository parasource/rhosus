package registry

import (
	"bytes"
	"context"
	"crypto/md5"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"io"
	"net"
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

type ServerConfig struct {
	Host      string
	Port      string
	MaxSizeMb int32
	BlockSize int64
	PageSize  int64
}

type Server struct {
	mu     sync.RWMutex
	Config ServerConfig

	http *http.Server

	shutdownC chan struct{}
	readyC    chan struct{}

	registry *Registry
}

func NewServer(r *Registry, conf ServerConfig) (*Server, error) {
	s := &Server{
		registry:  r,
		Config:    conf,
		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

	httpServer := &http.Server{
		Addr:    net.JoinHostPort(s.Config.Host, s.Config.Port),
		Handler: http.HandlerFunc(s.Handle),
		//TLSConfig:         nil,
		//ReadTimeout:       0,
		//ReadHeaderTimeout: 0,
		//WriteTimeout:      0,
		//IdleTimeout:       0,
		//MaxHeaderBytes:    0,
		//TLSNextProto:      nil,
		//ConnState:         nil,
		//ErrorLog:          nil,
		//BaseContext:       nil,
		//ConnContext:       nil,
	}

	s.http = httpServer

	return s, nil
}

func (s *Server) RunHTTP() {

	go func() {
		err := s.http.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logrus.Errorf("error listening: %v", err)
		}
	}()

	logrus.Infof("HTTP file server is up and running on %v", net.JoinHostPort(s.Config.Host, s.Config.Port))
	s.readyC <- struct{}{}

	if <-s.NotifyShutdown(); true {
		logrus.Infof("shutting down HTTP server")
		err := s.http.Shutdown(context.Background())
		if err != nil {
			logrus.Errorf("error occured while shutting down http server: %v", err)
		}
	}
}

func (s *Server) Shutdown() {
	close(s.shutdownC)
}

func (s *Server) NotifyShutdown() <-chan struct{} {
	return s.shutdownC
}

func (s *Server) NotifyReady() <-chan struct{} {
	return s.readyC
}

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "Rhosus file Server "+util.VERSION)
	//if r.Header.Get("Origin") != "" {
	//	w.Header().Set("Access-Control-Allow-Origin", "*")
	//	w.Header().Set("Access-Control-Allow-Credentials", "true")
	//}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPost, http.MethodPut:
		err := s.handlePostPut(w, r)
		if err != nil {
			logrus.Errorf("error uploading file: %v", err)
		}
	case http.MethodDelete:
		s.handleDelete(w, r)
	case http.MethodOptions:
		s.handleOptions(w, r)
	}
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {

	filePath := strings.Trim(r.URL.Path, "/")

	w.Header().Set("Accept-Ranges", "bytes")

	// Returns file

	err := s.registry.GetFileHandler(filePath, func(block *fs_pb.Block) {
		reader := io.NopCloser(bytes.NewReader(block.Data))
		n, err := io.CopyN(w, reader, int64(block.Len))
		if err != nil || n != int64(len(block.Data)) {
			logrus.Errorf("something went wrong: %v, %v - %v", err, n, len(block.Data))
		}
		//reader.Close()
	})
	if err != nil {
		switch err {
		case ErrNoSuchFileOrDirectory:
			w.WriteHeader(404)
			return
		default:
			logrus.Errorf("error getting blocks: %v", err)
		}
	}
}

func (s *Server) handlePostPut(w http.ResponseWriter, r *http.Request) error {

	var err error

	logrus.Infof(r.URL.String())

	multipartReader, err := r.MultipartReader()
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
	filePath := strings.Trim(r.URL.String(), "/")
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
	err = s.registry.RegisterFile(file)
	if err != nil {
		switch err {
		case ErrFileExists:
			w.WriteHeader(http.StatusConflict)
			return nil
		case ErrNoSuchFileOrDirectory:
			w.WriteHeader(http.StatusBadRequest)
			return nil
		default:
			logrus.Errorf("error registring file: %v", err)
			w.WriteHeader(500)
			w.Write([]byte("server error. see logs"))
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

	err = s.registry.TransportAndRegisterBlocks(file.Id, dataToTransfer, int(file.Replication))
	if err != nil {
		logrus.Errorf("error transporting blocks to node: %v", err)
	}

	w.Write([]byte("OK"))

	return err

	// Stores or Updates file
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}
