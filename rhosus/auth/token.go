package auth

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/storage"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/rs/zerolog/log"
	"time"
)

type TokenStore struct {
	storage *storage.Storage
}

func NewTokenStore(s *storage.Storage) (*TokenStore, error) {
	m := &TokenStore{
		storage: s,
	}
	go m.watchForTokensExpiration()

	return m, nil
}

func (m *TokenStore) watchForTokensExpiration() {
	ticker := tickers.SetTicker(time.Second * 5)
	defer tickers.ReleaseTicker(ticker)
	for {
		select {
		case <-ticker.C:
			tokens, err := m.storage.GetAllTokens()
			if err != nil {
				log.Error().Err(err).Msg("error getting all tokens in expiration watch")
				continue
			}

			for _, token := range tokens {
				if time.Now().Unix() > token.ValidUntil {
					err := m.storage.RevokeToken(token)
					if err != nil {
						log.Error().Err(err).Str("token", token.Token).
							Str("role_id", token.RoleID).
							Msg("error revoking expired token")
					}
				}
			}
		}
	}
}

func (m *TokenStore) CreateToken(roleID string, ttl time.Duration) (*control_pb.Token, error) {
	tokenStr := util.GenerateSecureToken(32)

	token := &control_pb.Token{
		RoleID:     roleID,
		Token:      tokenStr,
		ValidUntil: time.Now().Add(ttl).Unix(),
	}
	err := m.storage.StoreToken(token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (m *TokenStore) GetToken(token string) (*control_pb.Token, error) {
	return m.storage.GetToken(token)
}

func (m *TokenStore) RevokeToken(token string) error {
	t, err := m.storage.GetToken(token)
	if err != nil {
		return fmt.Errorf("error getting token from storage: %w", err)
	}

	return m.storage.RevokeToken(t)
}
