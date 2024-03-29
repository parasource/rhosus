/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package auth

// So here will be authentication centre, role management etc.

type LoginRequest struct {
	Username string
	Data     map[string]interface{}
}

type LoginResponse struct {
	Token   string `json:"token"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Authenticator is interface for user authentication
// It is expected that other methods will be available
// now it is just Login - so user is either authenticated, or it's not
type Authenticator interface {
	Login(req LoginRequest) (LoginResponse, error)
}
