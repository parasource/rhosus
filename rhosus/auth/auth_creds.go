package auth

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"time"
)

const (
	defaultTokenTTLm = 30
)

var (
	ErrInvalidPassword = errors.New("invalid password")
)

var _ Authenticator = &CredentialsAuth{}

type CredentialsAuth struct {
	roleManager  *RoleManager
	tokenManager *TokenManager
}

func NewCredentialsAuth(roleManager *RoleManager, tokenManager *TokenManager) (Authenticator, error) {
	return &CredentialsAuth{
		roleManager:  roleManager,
		tokenManager: tokenManager,
	}, nil
}

func (a *CredentialsAuth) Authenticate(req AuthenticationRequest) (AuthenticationResponse, error) {
	// Parameters check
	if req.Username == "" {
		return AuthenticationResponse{}, errors.New("'name' is a required parameter")
	}
	if _, ok := req.Data["password"]; !ok {
		return AuthenticationResponse{}, errors.New("'password' is a required parameter")
	}
	password, ok := req.Data["password"].(string)
	if !ok {
		return AuthenticationResponse{}, errors.New("invalid password parameter")
	}

	role, err := a.roleManager.GetRole(req.Username)
	if err != nil {
		return AuthenticationResponse{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(role.Password), []byte(password)); err != nil {
		token, err := a.tokenManager.CreateToken(role.ID, time.Minute*defaultTokenTTLm)
		if err != nil {
			return AuthenticationResponse{}, nil
		}

		return AuthenticationResponse{
			Token: token.Token,
		}, nil
	}

	return AuthenticationResponse{}, ErrInvalidPassword
}

func (a *CredentialsAuth) Authorise(req AuthorisationRequest) (AuthorisationResponse, error) {
	return AuthorisationResponse{}, nil
}
