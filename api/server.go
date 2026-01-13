package api

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

	"github.com/xdb-dev/xdb/store"
)

const (
	defaultReadHeaderTimeout = 15 * time.Second
	defaultShutdownTimeout   = 30 * time.Second
)

// ServerConfig holds configuration for the HTTP server.
type ServerConfig struct {
	Addr              string
	SocketPath        string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func (c *ServerConfig) readHeaderTimeout() time.Duration {
	if c.ReadHeaderTimeout == 0 {
		return defaultReadHeaderTimeout
	}
	return c.ReadHeaderTimeout
}

func (c *ServerConfig) shutdownTimeout() time.Duration {
	if c.ShutdownTimeout == 0 {
		return defaultShutdownTimeout
	}
	return c.ShutdownTimeout
}

// ServerBuilder constructs a Server with optional store handlers.
type ServerBuilder struct {
	cfg          *ServerConfig
	startTime    time.Time
	middlewares  xapi.MiddlewareStack
	errorHandler xapi.ErrorHandler

	schemaStore store.SchemaStore
	tupleStore  store.TupleStore
	recordStore store.RecordStore
	healthStore any
}

// NewServerBuilder creates a new ServerBuilder with the given configuration.
func NewServerBuilder(cfg *ServerConfig) *ServerBuilder {
	return &ServerBuilder{
		cfg:          cfg,
		startTime:    time.Now(),
		errorHandler: &StoreErrorHandler{},
	}
}

// WithSchemaStore configures the server to handle schema endpoints.
func (b *ServerBuilder) WithSchemaStore(s store.SchemaStore) *ServerBuilder {
	b.schemaStore = s
	return b
}

// WithTupleStore configures the server to handle tuple endpoints.
func (b *ServerBuilder) WithTupleStore(s store.TupleStore) *ServerBuilder {
	b.tupleStore = s
	return b
}

// WithRecordStore configures the server to handle record endpoints.
func (b *ServerBuilder) WithRecordStore(s store.RecordStore) *ServerBuilder {
	b.recordStore = s
	return b
}

// WithHealthStore configures the server to handle health endpoints.
// The store can be any type; if it implements store.HealthChecker, health checks will be performed.
func (b *ServerBuilder) WithHealthStore(s any) *ServerBuilder {
	b.healthStore = s
	return b
}

// WithMiddleware adds middleware to be applied to data endpoints.
func (b *ServerBuilder) WithMiddleware(m ...xapi.MiddlewareHandler) *ServerBuilder {
	b.middlewares = append(b.middlewares, m...)
	return b
}

// Build creates the Server with all configured handlers.
func (b *ServerBuilder) Build() (*Server, error) {
	if b.schemaStore == nil && b.tupleStore == nil && b.recordStore == nil && b.healthStore == nil {
		return nil, ErrNoStoresConfigured
	}

	s := &Server{
		cfg:       b.cfg,
		startTime: b.startTime,
		mux:       http.NewServeMux(),
	}

	b.registerHealthRoutes(s.mux)
	b.registerSchemaRoutes(s.mux)
	b.registerTupleRoutes(s.mux)
	b.registerRecordRoutes(s.mux)

	return s, nil
}

func (b *ServerBuilder) registerHealthRoutes(mux *http.ServeMux) {
	if b.healthStore == nil {
		return
	}

	healthAPI := NewHealthAPI(b.healthStore, b.startTime)
	health := xapi.NewEndpoint(
		xapi.EndpointFunc[HealthRequest, HealthResponse](healthAPI.GetHealth()),
		xapi.WithErrorHandler(b.errorHandler),
	)
	mux.Handle("GET /v1/health", health.Handler())
}

func (b *ServerBuilder) registerSchemaRoutes(mux *http.ServeMux) {
	if b.schemaStore == nil {
		return
	}

	schemaAPI := NewSchemaAPI(b.schemaStore)

	putSchema := xapi.NewEndpoint(
		xapi.EndpointFunc[PutSchemaRequest, PutSchemaResponse](schemaAPI.PutSchema()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	getSchema := xapi.NewEndpoint(
		xapi.EndpointFunc[GetSchemaRequest, GetSchemaResponse](schemaAPI.GetSchema()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	listSchemas := xapi.NewEndpoint(
		xapi.EndpointFunc[ListSchemasRequest, ListSchemasResponse](schemaAPI.ListSchemas()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	deleteSchema := xapi.NewEndpoint(
		xapi.EndpointFunc[DeleteSchemaRequest, DeleteSchemaResponse](schemaAPI.DeleteSchema()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)

	mux.Handle("PUT /v1/schemas", putSchema.Handler())
	mux.Handle("GET /v1/schemas/{uri...}", getSchema.Handler())
	mux.Handle("GET /v1/schemas", listSchemas.Handler())
	mux.Handle("DELETE /v1/schemas/{uri...}", deleteSchema.Handler())
}

func (b *ServerBuilder) registerTupleRoutes(mux *http.ServeMux) {
	if b.tupleStore == nil {
		return
	}

	tupleAPI := NewTupleAPI(b.tupleStore)

	putTuples := xapi.NewEndpoint(
		xapi.EndpointFunc[PutTuplesRequest, PutTuplesResponse](tupleAPI.PutTuples()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	getTuples := xapi.NewEndpoint(
		xapi.EndpointFunc[GetTuplesRequest, GetTuplesResponse](tupleAPI.GetTuples()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	deleteTuples := xapi.NewEndpoint(
		xapi.EndpointFunc[DeleteTuplesRequest, DeleteTuplesResponse](tupleAPI.DeleteTuples()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)

	mux.Handle("PUT /v1/tuples", putTuples.Handler())
	mux.Handle("GET /v1/tuples", getTuples.Handler())
	mux.Handle("DELETE /v1/tuples", deleteTuples.Handler())
}

func (b *ServerBuilder) registerRecordRoutes(mux *http.ServeMux) {
	if b.recordStore == nil {
		return
	}

	recordAPI := NewRecordAPI(b.recordStore)

	putRecords := xapi.NewEndpoint(
		xapi.EndpointFunc[PutRecordsRequest, PutRecordsResponse](recordAPI.PutRecords()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	getRecords := xapi.NewEndpoint(
		xapi.EndpointFunc[GetRecordsRequest, GetRecordsResponse](recordAPI.GetRecords()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)
	deleteRecords := xapi.NewEndpoint(
		xapi.EndpointFunc[DeleteRecordsRequest, DeleteRecordsResponse](recordAPI.DeleteRecords()),
		xapi.WithMiddleware(b.middlewares...),
		xapi.WithErrorHandler(b.errorHandler),
	)

	mux.Handle("PUT /v1/records", putRecords.Handler())
	mux.Handle("GET /v1/records", getRecords.Handler())
	mux.Handle("DELETE /v1/records", deleteRecords.Handler())
}

// Server is the XDB HTTP server.
type Server struct {
	cfg       *ServerConfig
	startTime time.Time
	mux       *http.ServeMux
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Run starts the server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	listeners, err := s.createListeners(ctx)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: s.cfg.readHeaderTimeout(),
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.shutdownTimeout())
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)

	if s.cfg.SocketPath != "" {
		removeSocketFile(s.cfg.SocketPath)
	}

	wg.Wait()
	return shutdownErr
}

func (s *Server) createListeners(ctx context.Context) ([]net.Listener, error) {
	var listeners []net.Listener

	if s.cfg.Addr != "" {
		l, err := createTCPListener(ctx, s.cfg.Addr)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, l)
		slog.Info("[SERVER] Starting TCP listener", "addr", s.cfg.Addr)
	}

	if s.cfg.SocketPath != "" {
		l, err := createUnixListener(ctx, s.cfg.SocketPath)
		if err != nil {
			for _, listener := range listeners {
				_ = listener.Close()
			}
			return nil, err
		}
		listeners = append(listeners, l)
		slog.Info("[SERVER] Starting Unix socket listener", "path", s.cfg.SocketPath)
	}

	return listeners, nil
}

// LoggingMiddleware logs HTTP requests with method, path, and duration.
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
