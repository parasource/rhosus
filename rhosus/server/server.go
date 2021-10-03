package server

import (
	"net/http"
	"strings"
)

type ServerConfig struct {
	Host      string
	Port      string
	MaxSizeMb int32
}

type Server struct {
	config ServerConfig
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

func (s *Server) handlePostPut(w http.ResponseWriter, r *http.Request) {

	// Stores or Updates file
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}
