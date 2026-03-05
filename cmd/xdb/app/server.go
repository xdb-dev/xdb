package app

import (
	"context"

	"github.com/gojekfarm/xtools/xapi"

	"github.com/xdb-dev/xdb/api"
)

// Server wraps the api.Server for use in the CLI application.
type Server struct {
	*api.Server
	cleanup []func() error
}

// NewServer creates a new Server with the given configuration.
func NewServer(cfg *Config) (*Server, error) {
	ss, err := initStoreFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	serverCfg := &api.ServerConfig{
		Addr:       cfg.Daemon.Addr,
		SocketPath: cfg.SocketPath(),
	}

	srv, err := api.NewServerBuilder(serverCfg).
		WithSchemaStore(ss.schema).
		WithTupleStore(ss.tuple).
		WithRecordStore(ss.record).
		WithHealthStore(ss.health).
		WithMiddleware(xapi.MiddlewareFunc(api.LoggingMiddleware)).
		Build()
	if err != nil {
		return nil, err
	}

	return &Server{Server: srv, cleanup: ss.cleanup}, nil
}

// Close releases resources held by the server's store backends.
func (s *Server) Close() error {
	for _, fn := range s.cleanup {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

// Run starts the server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	defer func() { _ = s.Close() }()
	return s.Server.Run(ctx)
}
