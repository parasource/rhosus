package api

import (
	"net/http"
	"strings"
	"time"
)

func (a *Api) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		if path != "sys/login" && path != "sys/create-test-user" {
			clientToken := r.Header.Get("X-Rhosus-Token")
			if clientToken == "" {
				rw.WriteHeader(http.StatusUnauthorized)
				rw.Write([]byte(ErrorMissingClientToken.Error()))
				return
			}

			token, err := a.tokenManager.GetToken(clientToken)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			if token == nil {
				rw.WriteHeader(http.StatusUnauthorized)
				rw.Write([]byte(ErrorInvalidClientToken.Error()))
				return
			}

			// Basically expired tokens are expected to
			// be removed by TokenStore special goroutine,
			// but will check additionally
			if token.Ttl+token.CreationTime < time.Now().Unix() {
				rw.WriteHeader(http.StatusUnauthorized)
				rw.Write([]byte(ErrorInvalidClientToken.Error()))
				return
			}
		}

		next.ServeHTTP(rw, r)
	})
}

func (a *Api) JsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")

		next.ServeHTTP(rw, r)
	})
}
