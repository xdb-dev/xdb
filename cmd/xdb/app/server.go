package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"github.com/gojekfarm/xtools/xapi"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

type Storer interface {
	driver.RepoDriver
}

type Server struct {
	cfg         *Config
	store       Storer
	middlewares xapi.MiddlewareStack
	mux         *http.ServeMux
}

func NewServer(cfg *Config) (*Server, error) {
	server := &Server{
		cfg: cfg,
	}

	err := errors.Join(
		server.initStore(),
		server.initMiddlewares(),
		server.registerRoutes(),
	)

	return server, err
}

func (s *Server) Run(ctx context.Context) error {
	slog.Info("[SERVER] Starting HTTP server", "addr", s.cfg.Addr)

	server := &http.Server{
		Addr:    s.cfg.Addr,
		Handler: s.mux,
	}

	go func(server *http.Server) {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}(server)

	<-ctx.Done()

	slog.Info("[SERVER] Shutting down HTTP server")
	return server.Shutdown(ctx)
}

func (s *Server) initStore() error {
	switch {
	case s.cfg.Store.SQLite != nil:
		slog.Info("[SERVER] Initializing SQLite store", "dir", s.cfg.Store.SQLite.Dir)

		store, err := xdbsqlite.New(*s.cfg.Store.SQLite)
		if err != nil {
			return err
		}
		s.store = store
	default:
		slog.Info("[SERVER] Initializing in-memory store")

		s.store = xdbmemory.New()

	}

	return nil
}

func (s *Server) initMiddlewares() error {
	s.middlewares = xapi.MiddlewareStack{
		xapi.MiddlewareFunc(LoggingMiddleware),
	}
	return nil
}

func (s *Server) registerRoutes() error {
	s.mux = http.NewServeMux()

	repoAPI := api.NewRepoAPI(s.store)

	makeRepo := xapi.NewEndpoint(
		xapi.EndpointFunc[api.MakeRepoRequest, api.MakeRepoResponse](repoAPI.MakeRepo()),
		xapi.WithMiddleware(s.middlewares...),
	)

	s.mux.Handle("POST /v1/repos", makeRepo.Handler())

	return nil
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)
		path := fmt.Sprintf("[HTTP] %s %s", r.Method, r.URL.Path)

		slog.Info(path, "duration", duration)
	})
}
