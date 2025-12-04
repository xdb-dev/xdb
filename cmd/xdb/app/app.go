// Package app wires together configurations, drivers, etc
// for the XDB CLI.
package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
)

type App struct {
	cfg     *Config
	cleanup []func() error // cleanup functions to be called on shutdown

	SchemaDriver driver.SchemaDriver
	TupleDriver  driver.TupleDriver
	RecordDriver driver.RecordDriver
}

func New(cfg *Config) (*App, error) {
	app := &App{cfg: cfg}

	err := errors.Join(
		app.initStore(),
	)

	return app, err
}

func (a *App) initStore() error {
	switch {
	case a.cfg.Store.SQLite != nil:
		slog.Info("[APP] SQLite store not yet fully implemented, using in-memory store")
		store := xdbmemory.New()
		a.SchemaDriver = store
		a.TupleDriver = store
		a.RecordDriver = store
	default:
		slog.Info("[APP] Initializing in-memory store")

		store := xdbmemory.New()
		a.SchemaDriver = store
		a.TupleDriver = store
		a.RecordDriver = store
	}

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	for _, cleanup := range a.cleanup {
		if err := cleanup(); err != nil {
			return err
		}
	}
	return nil
}
