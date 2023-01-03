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
	api_pb "github.com/parasource/rhosus/rhosus/pb/api"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"sync"
)

type Config struct {
	Address   string
	MaxSizeMb int32
	BlockSize int64
	PageSize  int64
}

type Api struct {
	mu sync.RWMutex

	Config   Config
	registry *registry.Registry
	http     *http.Server

	shutdownCh chan struct{}
	shutdown   bool

	encoder jsonpb.Marshaler
	decoder jsonpb.Unmarshaler
}

func NewApi(r *registry.Registry, conf Config) (*Api, error) {
	a := &Api{
		registry:   r,
		Config:     conf,
		shutdownCh: make(chan struct{}, 1),
		shutdown:   false,

		encoder: jsonpb.Marshaler{},
		decoder: jsonpb.Unmarshaler{},
	}

	httpServer := &http.Server{
		Addr:    conf.Address,
		Handler: http.HandlerFunc(a.Handle),
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
			logrus.Errorf("error listening: %v", err)
		}
	}()

	logrus.Infof("API server is up and running on %v", a.Config.Address)

	if <-a.NotifyShutdown(); true {
		logrus.Infof("shutting down API server")
		err := a.http.Shutdown(context.Background())
		if err != nil {
			logrus.Errorf("error occured while shutting down API server: %v", err)
		}
	}
}

func (a *Api) Handle(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Server", "Rhosus file Server "+util.VERSION)
	//if r.Header.Get("Origin") != "" {
	//	rw.Header().Set("Access-Control-Allow-Origin", "*")
	//	rw.Header().Set("Access-Control-Allow-Credentials", "true")
	//}

	var err error

	// We handle sys requests separately
	if strings.HasPrefix(strings.Trim(r.URL.Path, "/"), "sys") {
		err = a.HandleSys(rw, r)
		if err != nil {
			rw.WriteHeader(500)
			return
		}
	} else {
		switch r.Method {
		case http.MethodGet:
			a.handleGet(rw, r)
		case http.MethodPost, http.MethodPut:
			err := a.handlePostPut(rw, r)
			if err != nil {
				logrus.Errorf("error uploading file: %v", err)
			}
		case http.MethodDelete:
			a.handleDelete(rw, r)
		case http.MethodOptions:
			a.handleOptions(rw, r)
		}
	}
}

func (a *Api) HandleSys(rw http.ResponseWriter, r *http.Request) error {
	var body []byte

	switch strings.Trim(r.URL.Path, "/") {
	case "sys/mkdir":
		_, err := r.Body.Read(body)
		if err != nil {
			logrus.Errorf("error reading request body: %v", err)
			return err
		}

		var msg api_pb.MakeDirRequest
		err = a.decoder.Unmarshal(r.Body, &msg)
		if err != nil {
			logrus.Errorf("error unmarshaling sys request: %v", err)
			return err
		}

		res, err := a.registry.HandleMakeDir(&msg)
		if err != nil {
			return err
		}

		rw.WriteHeader(200)
		a.encoder.Marshal(rw, res)

		return nil
	case "sys/rm":
		_, err := r.Body.Read(body)
		if err != nil {
			logrus.Errorf("error reading request body: %v", err)
			return err
		}

		var msg api_pb.RemoveRequest
		err = a.decoder.Unmarshal(r.Body, &msg)
		if err != nil {
			logrus.Errorf("error unmarshaling sys request: %v", err)
			return err
		}

		res, err := a.registry.HandleRemoveFileOrPath(&msg)
		if err != nil {
			return err
		}

		rw.WriteHeader(200)
		a.encoder.Marshal(rw, res)

		return nil
	case "sys/list":
		_, err := r.Body.Read(body)
		if err != nil {
			logrus.Errorf("error reading request body: %v", err)
			return err
		}

		var msg api_pb.ListRequest
		err = a.decoder.Unmarshal(r.Body, &msg)
		if err != nil {
			logrus.Errorf("error unmarshaling sys request: %v", err)
			return err
		}

		res, err := a.registry.HandleList(&msg)
		if err != nil {
			return err
		}

		rw.WriteHeader(200)
		a.encoder.Marshal(rw, res)

		return nil
	case "sys/hierarchy":

	}

	return nil
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
