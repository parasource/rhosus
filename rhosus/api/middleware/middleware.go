package middleware

import "net/http"

type Middleware interface {
	Check(rw http.ResponseWriter, r *http.Request)
}
