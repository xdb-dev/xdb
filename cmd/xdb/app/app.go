// Package app wires together configurations, drivers, etc
// for the XDB CLI.
package app

import (
	"context"
	"errors"

	"log/slog"

	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

type App struct {
	cfg     *Config
	cleanup []func() error // cleanup functions to be called on shutdown

	RepoDriver driver.RepoDriver
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
		slog.Info("[APP] Initializing SQLite store", "path", a.cfg.Store.SQLite.Path)

		store, err := xdbsqlite.New(*a.cfg.Store.SQLite)
		if err != nil {
			return err
		}
		a.RepoDriver = store

		a.registerCleanup(func() error {
			return store.Close()
		})
	default:
		slog.Info("[APP] Initializing in-memory store")

		store := xdbmemory.New()
		a.RepoDriver = store
	}

	return nil
}

func (a *App) registerCleanup(cleanup func() error) {
	a.cleanup = append(a.cleanup, cleanup)
}

func (a *App) Shutdown(ctx context.Context) error {
	for _, cleanup := range a.cleanup {
		if err := cleanup(); err != nil {
			return err
		}
	}
	return nil
}
