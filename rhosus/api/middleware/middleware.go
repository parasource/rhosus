package middleware

import "net/http"

type Middleware interface {
	Check(next http.HandlerFunc) http.HandlerFunc
}
