/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package api

import (
	"context"
	"github.com/golang/protobuf/jsonpb"
	"github.com/parasource/rhosus/rhosus/auth"
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/rs/zerolog/log"
	"net/http"
	"sync"
)

type Config struct {
	Address   string
	MaxSizeMb int32
	BlockSize int64
	PageSize  int64

	AuthMethods map[string]auth.Authenticator
}

type Api struct {
	mu sync.RWMutex

	Config   Config
	registry *registry.Registry
	http     *http.Server

	tokenManager *auth.TokenStore

	shutdownCh chan struct{}
	shutdown   bool

	encoder jsonpb.Marshaler
	decoder jsonpb.Unmarshaler
}

func NewApi(r *registry.Registry, tokenManager *auth.TokenStore, conf Config) (*Api, error) {
	a := &Api{
		registry:   r,
		Config:     conf,
		shutdownCh: make(chan struct{}, 1),
		shutdown:   false,

		encoder: jsonpb.Marshaler{},
		decoder: jsonpb.Unmarshaler{},

		tokenManager: tokenManager,
	}

	httpServer := &http.Server{
		Addr: conf.Address,
		//Handler: http.HandlerFunc(a.Handle),
		Handler: a.Router(),
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

	a.http = httpServer

	return a, nil
}

func (a *Api) Run() {
	go func() {
		err := a.http.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("error listening api")
		}
	}()

	log.Info().Str("address", a.Config.Address).Msg("API server is up and running")

	if <-a.NotifyShutdown(); true {
		log.Info().Msg("shutting down API server")
		err := a.http.Shutdown(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("error occurred while shutting down API server")
		}
	}
}

func (a *Api) HandleFilesystem(rw http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleGet(rw, r)
	case http.MethodPost, http.MethodPut:
		err := a.handlePostPut(rw, r)
		if err != nil {
			log.Error().Err(err).Msg("error handling Post/Put operation")
		}
	case http.MethodDelete:
		a.handleDelete(rw, r)
	case "LIST":
		err := a.handleList(rw, r)
		if err != nil {
			log.Error().Err(err).Msg("error handling LIST operation")
		}
	case http.MethodOptions:
		a.handleOptions(rw, r)
	}
}

func prepareLoginRequest(req api_pb.LoginRequest) auth.LoginRequest {
	name := req.Data["name"]
	delete(req.Data, "name")
	data := make(map[string]interface{}, len(req.Data))
	for k, v := range req.Data {
		data[k] = v
	}
	return auth.LoginRequest{
		Username: name,
		Data:     data,
	}
}

func (a *Api) handleGet(rw http.ResponseWriter, r *http.Request) error {
	err := a.registry.HandleGetFile(rw, r)
	return err
}

func (a *Api) handlePostPut(rw http.ResponseWriter, r *http.Request) error {
	err := a.registry.HandlePutFile(rw, r)
	return err
}

func (a *Api) handleDelete(rw http.ResponseWriter, r *http.Request) error {
	err := a.registry.HandleDeleteFile(rw, r)
	return err
}

func (a *Api) handleList(rw http.ResponseWriter, r *http.Request) error {
	err := a.registry.HandleList(rw, r)
	return err
}

func (a *Api) handleOptions(w http.ResponseWriter, r *http.Request) {

	// Deletes file
}

func (a *Api) NotifyShutdown() <-chan struct{} {
	return a.shutdownCh
}

func (a *Api) Shutdown() {
	a.mu.RLock()
	if a.shutdown {
		a.mu.RUnlock()
		return
	}
	a.mu.RUnlock()

	close(a.shutdownCh)
}
