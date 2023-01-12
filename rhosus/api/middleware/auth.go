package middleware

import (
	"github.com/parasource/rhosus/rhosus/auth"
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

func (a *Auth) Check(rw http.ResponseWriter, r *http.Request) {

}
