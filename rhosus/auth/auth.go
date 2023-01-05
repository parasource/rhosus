/*
 * Copyright (c) 2022.
 * Licensed to the Parasource Foundation under one or more contributor license agreements.  See the NOTICE file distributed with this work for additional information regarding copyright ownership.  The Parasource licenses this file to you under the Parasource License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://www.parasource.org/licenses/LICENSE-2.0 Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package auth

// So here will be authentication centre, role management etc.

type AuthenticationRequest struct {
	Username string
	Data     map[string]interface{}
}

type AuthenticationResponse struct {
	Token string `json:"token"`
}

type AuthorisationRequest struct {
}

type AuthorisationResponse struct {
}

type Authenticator interface {
	Authenticate(req AuthenticationRequest) (AuthenticationResponse, error)
	Authorise(req AuthorisationRequest) (AuthorisationResponse, error)
}
