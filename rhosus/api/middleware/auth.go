package middleware

import (
	"github.com/parasource/rhosus/rhosus/auth"
	"github.com/rs/zerolog/log"
	"net/http"
)

type Auth struct {
	authenticator auth.Authenticator
}

func NewAuthMiddleware(a auth.Authenticator) Middleware {
	return &Auth{
		authenticator: a,
	}
}

func (a *Auth) Check(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		res, err := a.authenticator.Authorize(auth.AuthorizationRequest{
			Path:   r.URL.Path,
			Method: auth.Method(r.Method),
			Token:  r.Header.Get("X-Rhosus-Token"),
		})
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msg("error authorizing client")
			return
		}

		if !res.Success {
			rw.WriteHeader(http.StatusUnauthorized)
			log.Debug().Msg("unsuccessful authorization attempt")
			return
		}

		next(rw, r)
	}
}
