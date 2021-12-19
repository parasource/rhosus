package file_server

import (
	"context"
	"github.com/parasource/rhosus/rhosus/sys"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
	"sync"
)

type ServerConfig struct {
	Host      string
	Port      string
	MaxSizeMb int32
}

type Server struct {
	mu     sync.RWMutex
	Config ServerConfig

	http *http.Server

	shutdownCh chan struct{}
	readyCh    chan struct{}

	RegistryAddFunc    func(dir string, name string, owner string, group string, timestamp int64, size uint64, data []byte) (*sys.File, error)
	RegistryDeleteFunc func(dir string, name string) error
}

func NewServer(conf ServerConfig) (*Server, error) {
	s := &Server{
		Config:  conf,
		readyCh: make(chan struct{}, 1),
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

	//go s.RunHTTP()

	return s, nil
}

func (s *Server) RunHTTP() {

	go func() {
		err := s.http.ListenAndServe()
		if err != nil {
			logrus.Errorf("error listening: %v", err)
		}
	}()

	logrus.Infof("HTTP file server is up and running on %v", net.JoinHostPort(s.Config.Host, s.Config.Port))
	s.readyCh <- struct{}{}

	for {
		select {
		case <-s.NotifyShutdown():
			err := s.http.Shutdown(context.Background())
			if err != nil {
				logrus.Errorf("error occured while shutting down http server: %v", err)
			}
		}
	}
}

func (s *Server) SendShutdownSignal() {
	s.shutdownCh <- struct{}{}
}

func (s *Server) NotifyShutdown() <-chan struct{} {
	return s.shutdownCh
}

func (s *Server) NotifyReady() <-chan struct{} {
	return s.readyCh
}

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	//w.Header().Set("Server", "Rhosus File Server "+util.VERSION)
	//if r.Header.Get("Origin") != "" {
	//	w.Header().Set("Access-Control-Allow-Origin", "*")
	//	w.Header().Set("Access-Control-Allow-Credentials", "true")
	//}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r)
	case http.MethodPost, http.MethodPut:
		s.handlePostPut(w, r)
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
	defer s.mu.Unlock()

	s.RegistryAddFunc = fun
}

func (s *Server) SetRegistryDeleteFunc(fun func(dir string, name string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.RegistryDeleteFunc = fun
}

func (s *Server) handlePostPut(w http.ResponseWriter, r *http.Request) {

	// Stores or Updates file
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}
