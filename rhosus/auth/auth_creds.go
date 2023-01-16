package auth

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"time"
)

const (
	defaultTokenTTLm = 30
)

var _ Authenticator = &CredentialsAuth{}

type CredentialsAuth struct {
	roleManager  *RoleManager
	tokenManager *TokenManager
}

func NewCredentialsAuth(roleManager *RoleManager, tokenManager *TokenManager) Authenticator {
	return &CredentialsAuth{
		roleManager:  roleManager,
		tokenManager: tokenManager,
	}
}

// Login method is used to get token
// After that, client must embed this token in all requests
func (a *CredentialsAuth) Login(req LoginRequest) (LoginResponse, error) {
	// Parameters check
	if req.Username == "" {
		return LoginResponse{
			Success: false,
			Message: "'name' is a required parameter",
		}, nil
	}
	if _, ok := req.Data["password"]; !ok {
		return LoginResponse{
			Success: false,
			Message: "'password' is a required parameter",
		}, nil
	}
	password, ok := req.Data["password"].(string)
	if !ok {
		return LoginResponse{}, errors.New("invalid password parameter")
	}

	role, err := a.roleManager.GetRole(req.Username)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("error getting role: %w", err)
	}
	if role == nil {
		return LoginResponse{
			Success: false,
			Message: "role is not found",
		}, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(role.Password), []byte(password)); err == nil {
		token, err := a.tokenManager.CreateToken(role.ID, time.Minute*defaultTokenTTLm)
		if err != nil {
			return LoginResponse{}, fmt.Errorf("error populating token: %w", err)
		}

		return LoginResponse{
			Token:   token.Token,
			Success: true,
		}, nil
	}

	return LoginResponse{
		Success: false,
		Message: "invalid password",
	}, nil
}

// Authorize method is used to check if token is
// not expired and if user has permissions to access
// this path. Currently, it does not check neither path nor method
// as the permissions are not implemented.
func (a *CredentialsAuth) Authorize(req AuthorizationRequest) (AuthorizationResponse, error) {
	if req.Token == "" {
		return AuthorizationResponse{}, errors.New("'token' is a required parameter")
	}

	token, err := a.tokenManager.GetToken(req.Token)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("error getting token: %w", err)
	}
	if token == nil {
		return AuthorizationResponse{}, errors.New("token is nil")
	}

	// if token is expired
	if token.ValidUntil < time.Now().Unix() {
		return AuthorizationResponse{
			Success: false,
		}, nil
	}

	role, err := a.roleManager.GetRoleById(token.RoleID)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("error getting role")
	}

	return AuthorizationResponse{
		Role:    *role,
		Success: true,
	}, nil
}
