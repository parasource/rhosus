package file_server

import (
	"bytes"
	"context"
	"crypto/md5"
	"github.com/parasource/rhosus/rhosus/sys"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"path"
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

	RegistryAdd    func(dir string, name string, owner string, group string, timestamp int64, size uint64, data []byte) (*sys.File, error)
	RegistryDelete func(dir string, name string) error
}

func NewServer(conf ServerConfig) (*Server, error) {
	s := &Server{
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

	select {
	case <-s.NotifyShutdown():
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
	w.Header().Set("Server", "Rhosus File Server "+util.VERSION)
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

	path := r.URL.Path
	isForDirectory := strings.HasSuffix(path, "/")
	if isForDirectory && len(path) > 1 {
		path = path[:len(path)-1]
	}

	w.Header().Set("Accept-Ranges", "bytes")

	// Returns file

	w.Write([]byte("you are a leader, neo"))
}

func (s *Server) SetRegistryAddFunc(fun func(dir string, name string, owner string, group string, timestamp int64, size uint64, data []byte) (*sys.File, error)) {
	s.mu.Lock()
	s.RegistryAdd = fun
	s.mu.Unlock()
}

func (s *Server) SetRegistryDeleteFunc(fun func(dir string, name string) error) {
	s.mu.Lock()
	s.RegistryDelete = fun
	s.mu.Unlock()
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

	fileName := part1.FileName()
	if fileName != "" {
		fileName = path.Base(fileName)
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

	blockSize := s.Config.BlockSize

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
			chunkOffset += dataSize
			smallContent := make([]byte, dataSize)
			bytesBuffer.Read(smallContent)
			bufPool.Put(bytesBuffer)
			atomic.AddInt64(&bytesBufferCounter, -1)
			bytesBufferLimitCond.Signal()
			break
		}

		//logrus.Info(string(bytesBuffer.Bytes()))

		wg.Add(1)
		counter := 1
		go func(offset int64) {
			defer func() {
				bufPool.Put(bytesBuffer)
				atomic.AddInt64(&bytesBufferCounter, -1)
				bytesBufferLimitCond.Signal()
				wg.Done()
			}()

			dataReader := util.NewBytesReader(bytesBuffer.Bytes())

			var data = dataReader.Bytes

			logrus.Info(string(data))
			logrus.Infof("------------- %v", counter)

			counter++

		}(chunkOffset)

		chunkOffset = chunkOffset + dataSize

		// if last chunk was not at full chunk size, but already exhausted the reader
		if dataSize < blockSize {
			break
		}
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
