package app

import (
	"context"
	"log/slog"

	"github.com/gojekfarm/xtools/xapi"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// Server wraps the api.Server for use in the CLI application.
type Server struct {
	*api.Server
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfg *Config) (*Server, error) {
	slog.Info("[SERVER] Initializing in-memory store")
	memStore := xdbmemory.New()

	serverCfg := &api.ServerConfig{
		Addr:       cfg.Daemon.Addr,
		SocketPath: cfg.SocketPath(),
	}

	srv, err := api.NewServerBuilder(serverCfg).
		WithSchemaStore(memStore).
		WithTupleStore(memStore).
		WithRecordStore(memStore).
		WithHealthStore(memStore).
		WithMiddleware(xapi.MiddlewareFunc(api.LoggingMiddleware)).
		Build()
	if err != nil {
		return nil, err
	}

	return &Server{Server: srv}, nil
}

// Run starts the server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	return s.Server.Run(ctx)
}
