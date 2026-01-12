package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gojekfarm/xtools/errors"
	"github.com/gojekfarm/xtools/xapi"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

type Storer interface {
	store.SchemaStore
	store.TupleStore
	store.RecordStore
}

type Server struct {
	cfg         *Config
	store       Storer
	startTime   time.Time
	middlewares xapi.MiddlewareStack
	mux         *http.ServeMux
}

func NewServer(cfg *Config) (*Server, error) {
	server := &Server{
		cfg:       cfg,
		startTime: time.Now(),
	}

	err := errors.Join(
		server.initStore(),
		server.initMiddlewares(),
		server.registerRoutes(),
	)

	return server, err
}

func (s *Server) Run(ctx context.Context) error {
	listeners, err := s.createListeners(ctx)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: 15 * time.Second,
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(listeners))

	for _, listener := range listeners {
		wg.Add(1)
		go func(l net.Listener) {
			defer wg.Done()
			err := server.Serve(l)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				errChan <- err
			}
		}(listener)
	}

	select {
	case <-ctx.Done():
		slog.Info("[SERVER] Shutting down HTTP server")
	case err := <-errChan:
		slog.Error("[SERVER] Listener error, shutting down", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)

	if s.cfg.Daemon.Socket != "" {
		removeSocketFile(s.cfg.SocketPath())
	}

	wg.Wait()
	return shutdownErr
}

func (s *Server) createListeners(ctx context.Context) ([]net.Listener, error) {
	var listeners []net.Listener

	if s.cfg.Daemon.Addr != "" {
		l, err := createTCPListener(ctx, s.cfg.Daemon.Addr)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, l)
		slog.Info("[SERVER] Starting TCP listener", "addr", s.cfg.Daemon.Addr)
	}

	if s.cfg.Daemon.Socket != "" {
		socketPath := s.cfg.SocketPath()
		l, err := createUnixListener(ctx, socketPath)
		if err != nil {
			for _, listener := range listeners {
				_ = listener.Close()
			}
			return nil, err
		}
		listeners = append(listeners, l)
		slog.Info("[SERVER] Starting Unix socket listener", "path", socketPath)
	}

	return listeners, nil
}

func (s *Server) initStore() error {
	slog.Info("[SERVER] Initializing in-memory store")
	s.store = xdbmemory.New()

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

	// Health endpoint (no middleware for fast response)
	healthAPI := api.NewHealthAPI(s.store, s.startTime)
	health := xapi.NewEndpoint(
		xapi.EndpointFunc[api.HealthRequest, api.HealthResponse](healthAPI.GetHealth()),
	)
	s.mux.Handle("GET /v1/health", health.Handler())

	schemaAPI := api.NewSchemaAPI(s.store)

	putSchema := xapi.NewEndpoint(
		xapi.EndpointFunc[api.PutSchemaRequest, api.PutSchemaResponse](schemaAPI.PutSchema()),
		xapi.WithMiddleware(s.middlewares...),
	)
	getSchema := xapi.NewEndpoint(
		xapi.EndpointFunc[api.GetSchemaRequest, api.GetSchemaResponse](schemaAPI.GetSchema()),
		xapi.WithMiddleware(s.middlewares...),
	)
	listSchemas := xapi.NewEndpoint(
		xapi.EndpointFunc[api.ListSchemasRequest, api.ListSchemasResponse](schemaAPI.ListSchemas()),
		xapi.WithMiddleware(s.middlewares...),
	)
	deleteSchema := xapi.NewEndpoint(
		xapi.EndpointFunc[api.DeleteSchemaRequest, api.DeleteSchemaResponse](schemaAPI.DeleteSchema()),
		xapi.WithMiddleware(s.middlewares...),
	)

	s.mux.Handle("PUT /v1/schemas", putSchema.Handler())
	s.mux.Handle("GET /v1/schemas/{uri...}", getSchema.Handler())
	s.mux.Handle("GET /v1/schemas", listSchemas.Handler())
	s.mux.Handle("DELETE /v1/schemas/{uri...}", deleteSchema.Handler())

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

func createTCPListener(ctx context.Context, addr string) (net.Listener, error) {
	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP listener on %s: %w", addr, err)
	}
	return listener, nil
}

func createUnixListener(ctx context.Context, socketPath string) (net.Listener, error) {
	info, err := os.Stat(socketPath)
	if err == nil {
		if info.Mode()&os.ModeSocket != 0 {
			slog.Warn("[SERVER] Removing existing socket file", "path", socketPath)
			if err := os.Remove(socketPath); err != nil {
				return nil, errors.Wrap(err, "path", socketPath)
			}
		} else {
			return nil, errors.Wrap(ErrInvalidSocketFile, "path", socketPath)
		}
	} else if !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "path", socketPath)
	}

	lc := &net.ListenConfig{}
	listener, err := lc.Listen(ctx, "unix", socketPath)
	if err != nil {
		return nil, errors.Wrap(err, "path", socketPath)
	}

	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = listener.Close()
		return nil, errors.Wrap(err, "path", socketPath)
	}

	return listener, nil
}

func removeSocketFile(socketPath string) {
	info, err := os.Stat(socketPath)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		slog.Warn("[SERVER] Failed to check socket file for cleanup", "path", socketPath, "error", err)
		return
	}

	if info.Mode()&os.ModeSocket != 0 {
		if err := os.Remove(socketPath); err != nil {
			slog.Warn("[SERVER] Failed to remove socket file", "path", socketPath, "error", err)
		}
	}
}
