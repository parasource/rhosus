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
