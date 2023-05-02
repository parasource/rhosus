package api

import (
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
)

func (a *Api) Router() http.Handler {
	r := mux.NewRouter()

	// system routes
	sysr := r.PathPrefix("/sys").Subrouter()
	sysr.Use(a.JsonMiddleware)
	r.Use(a.AuthMiddleware)

	sysr.HandleFunc("/login", a.handleLogin).Methods(http.MethodPost)
	sysr.HandleFunc("/mkdir", a.handleMakeDir).Methods(http.MethodPost)
	sysr.HandleFunc("/create-test-user", a.handleCreateTestUser).Methods(http.MethodPost)

	sysr.HandleFunc("/policies/{name}", a.handleCreateUpdatePolicy).Methods(http.MethodPost, http.MethodPut)
	sysr.HandleFunc("/policies/{name}", a.handleGetPolicy).Methods(http.MethodGet)
	sysr.HandleFunc("/policies/{name}", a.handleDeletePolicy).Methods(http.MethodDelete)
	sysr.HandleFunc("/policies", a.handleListPolicies).Methods("LIST")

	sysr.HandleFunc("/tokens/create", a.handleCreateToken).Methods(http.MethodPost)
	sysr.HandleFunc("/tokens/{accessor}", a.handleRevokeToken).Methods(http.MethodDelete)
	sysr.HandleFunc("/tokens/{accessor}", a.handleGetToken).Methods(http.MethodGet)
	sysr.HandleFunc("/tokens", a.handleListTokens).Methods("LIST")

	r.PathPrefix("/").HandlerFunc(a.HandleFilesystem)

	cors.AllowAll()
	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			"LIST",
		},
		AllowedHeaders: []string{"*"},
		Debug:          true,
	}).Handler(r)
}
