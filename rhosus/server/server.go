package server

import (
	"github.com/parasource/rhosus/rhosus/storage"
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
	config ServerConfig

	RegistryAddFunc    func(dir string, name string, owner string, group string, data []byte) (*storage.StoragePlacementInfo, error)
	RegistryDeleteFunc func(dir string, name string) error
}

func NewServer(conf ServerConfig) (*Server, error) {
	s := &Server{
		config: conf,
	}
	return s, nil
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
}

func (s *Server) SetRegistryAddFunc(fun func(dir string, name string, owner string, group string, data []byte) (*storage.StoragePlacementInfo, error)) {
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
