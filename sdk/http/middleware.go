package znhttp

import (
	"net/http"
	"time"

	"github.com/blixenkrone/sdk/logger"
)

type Middleware struct {
	serviceName string
	log         logger.Logger
}

func NewHTTPMiddleware(
	serviceName string,
	log logger.Logger,
) Middleware {
	return Middleware{serviceName, log}
}

type MiddlewareFn func(next http.Handler) http.Handler

func (m Middleware) LogRoutes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := m.log.WithContext(r.Context())
		l.Debugf("http request received: %s -> %s", r.Method, r.URL)
		n := time.Now()
		next.ServeHTTP(w, r)
		l.Debugf("http request processed: %s -> %s took %s", r.Method, r.URL, time.Since(n).String())
	})
}
