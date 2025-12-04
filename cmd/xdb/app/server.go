package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gojekfarm/xtools/xapi"
	"log/slog"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
)

type Storer interface {
	driver.SchemaDriver
	driver.TupleDriver
	driver.RecordDriver
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
		slog.Info("[SERVER] Initializing SQLite store not yet fully supported, using in-memory store")
		s.store = xdbmemory.New()
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
