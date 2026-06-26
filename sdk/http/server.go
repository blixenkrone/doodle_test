package znhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/blixenkrone/sdk/logger"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

type routerEngine interface {
	http.Handler
	HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route
	Use(mwf ...mux.MiddlewareFunc)
	PathPrefix(tpl string) *mux.Route
}

type route struct {
	fn      http.HandlerFunc
	methods []string
}

type Server struct {
	serviceName string
	logger      logger.Logger
	srv         *http.Server
	router      routerEngine
	routes      map[routeProtector]route
	middleware  []MiddlewareFn
}

func NewServer(
	logger logger.Logger,
	handler routerEngine,
	addr, serviceName string,
) *Server {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       time.Second * 30,
		ReadHeaderTimeout: 0,
		WriteTimeout:      time.Second * 30,
		IdleTimeout:       time.Second * 30,
		MaxHeaderBytes:    1 << 20,
	}

	routes := make(map[routeProtector]route)
	var middleware []MiddlewareFn
	return &Server{serviceName, logger, srv, handler, routes, middleware}
}

func (s *Server) ListenAndServe() error {
	s.logger.Infof("starting %s server on %s", s.serviceName, s.srv.Addr)
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

type routeProtector struct {
	route  string
	method string
}

func (s *Server) AddRoute(endpoint string, fn http.HandlerFunc, methods ...string) {
	slices.Sort(methods)
	s.routes[routeProtector{
		route:  endpoint,
		method: strings.Join(methods, "-"), // To ensure that we can apply identical routes with different methods to different http.HandlerFuncs
	}] = route{fn, methods}
}

// Consider adding logic for path scoped middleware later
func (s *Server) Use(mw ...MiddlewareFn) {
	s.middleware = append(s.middleware, mw...)
}

// Must be called before serving application. Consider adding a check for this.
func (s *Server) RegisterRoutes() error {
	// TODO: Implement compression
	// TODO: CORS allow for all origins should be more restrictive
	var errs []error
	c := cors.AllowAll()

	// Swagger handler requires path matching on wildcard routes
	s.router.PathPrefix("/docs/").Handler(httpSwagger.Handler()).Methods(http.MethodGet)
	s.AddRoute("/health", s.health(), http.MethodGet)
	s.AddRoute("/", s.health(), http.MethodGet)
	s.AddRoute("/ready", s.ready(), http.MethodGet)
	s.Use(c.Handler)

	for _, fn := range s.middleware {
		s.router.Use(mux.MiddlewareFunc(fn))
	}

	for path, route := range s.routes {
		if len(route.methods) <= 0 {
			errs = append(errs, fmt.Errorf("route %s has 0 methods attached", path))
			continue
		}
		s.router.HandleFunc(path.route, route.fn).Methods(route.methods...)
	}
	return errors.Join(errs...)
}

// @Summary Container Liveness Probe.
// @Description Probe verifying that the container has started.
// @Success 200
// @Failure 500
// @Router /health [get]
// @Tags probe
func (s *Server) health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

// @Summary Container Readiness Probe.
// @Description Probe verifying that the app is fully ready to serve requests.
// @Success 200
// @Failure 500 {object} znhttp.HTTPError
// @Produce json
// @Router /ready [get]
// @Tags probe
func (s *Server) ready() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
