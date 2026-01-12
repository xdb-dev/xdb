// Package app wires together configurations, drivers, etc
// for the XDB CLI.
package app

import (
	"context"
	"log/slog"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

type App struct {
	cfg     *Config
	cleanup []func() error // cleanup functions to be called on shutdown

	SchemaDriver store.SchemaStore
	TupleDriver  store.TupleStore
	RecordDriver store.RecordStore
}

func New(cfg *Config) (*App, error) {
	app := &App{cfg: cfg}

	err := errors.Join(
		app.initStore(),
	)

	return app, err
}

func (a *App) initStore() error {
	slog.Info("Initializing in-memory store")

	st := xdbmemory.New()
	a.SchemaDriver = st
	a.TupleDriver = st
	a.RecordDriver = st

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

// ListOptions configures list operations.
type ListOptions struct {
	Limit  int
	Offset int
}

// GetByURI retrieves a resource (tuple, record, or schema) by URI.
func (a *App) GetByURI(ctx context.Context, uri *core.URI) (any, error) {
	switch {
	case uri.Attr() != nil:
		// Fetch specific tuple
		tuples, notFound, err := a.TupleDriver.GetTuples(ctx, []*core.URI{uri})
		if err != nil {
			return nil, errors.Wrap(err, "uri", uri.String())
		}
		if len(notFound) > 0 {
			return nil, errors.Wrap(ErrTupleNotFound, "uri", uri.String())
		}
		if len(tuples) == 0 {
			return nil, errors.Wrap(ErrNoTuplesReturned, "uri", uri.String())
		}
		return tuples[0], nil

	case uri.ID() != nil:
		// Fetch record
		records, notFound, err := a.RecordDriver.GetRecords(ctx, []*core.URI{uri})
		if err != nil {
			return nil, errors.Wrap(err, "uri", uri.String())
		}
		if len(notFound) > 0 {
			return nil, errors.Wrap(ErrRecordNotFound, "uri", uri.String())
		}
		if len(records) == 0 {
			return nil, errors.Wrap(ErrNoRecordsReturned, "uri", uri.String())
		}
		return records[0], nil

	case uri.Schema() != nil:
		// Fetch schema
		schema, err := a.SchemaDriver.GetSchema(ctx, uri)
		if err != nil {
			return nil, errors.Wrap(err, "uri", uri.String())
		}
		return schema, nil

	default:
		return nil, errors.Wrap(ErrURIMustSpecifySchema, "uri", uri.String())
	}
}

// PutRecord creates or updates a record.
func (a *App) PutRecord(ctx context.Context, record *core.Record) error {
	slog.Debug("Putting record", "uri", record.URI().String(), "tuples", len(record.Tuples()))

	err := a.RecordDriver.PutRecords(ctx, []*core.Record{record})
	if err != nil {
		return errors.Wrap(err, "uri", record.URI().String())
	}
	return nil
}

// ListByURI lists resources matching the URI pattern.
func (a *App) ListByURI(ctx context.Context, uri *core.URI, opts ListOptions) (any, error) {
	if uri.Schema() == nil {
		// List schemas in namespace
		schemas, err := a.SchemaDriver.ListSchemas(ctx, uri)
		if err != nil {
			return nil, errors.Wrap(err, "uri", uri.String())
		}
		return paginate(schemas, opts.Limit, opts.Offset), nil
	}

	// List records in schema - not yet supported by store layer
	return nil, ErrListingRecordsNotImplemented
}

// RemoveByURI deletes a resource (tuple, record, or schema) by URI.
func (a *App) RemoveByURI(ctx context.Context, uri *core.URI) error {
	switch {
	case uri.Attr() != nil:
		// Delete tuple
		err := a.TupleDriver.DeleteTuples(ctx, []*core.URI{uri})
		if err != nil {
			return errors.Wrap(err, "uri", uri.String())
		}
		return nil

	case uri.ID() != nil:
		// Delete record
		err := a.RecordDriver.DeleteRecords(ctx, []*core.URI{uri})
		if err != nil {
			return errors.Wrap(err, "uri", uri.String())
		}
		return nil

	case uri.Schema() != nil:
		// Delete schema
		err := a.SchemaDriver.DeleteSchema(ctx, uri)
		if err != nil {
			return errors.Wrap(err, "uri", uri.String())
		}
		return nil

	default:
		return errors.Wrap(ErrURIMustSpecifySchema, "uri", uri.String())
	}
}

// paginate applies limit/offset to a slice of schemas.
func paginate[T any](items []T, limit, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}

	start := offset
	end := len(items)

	if limit > 0 && start+limit < end {
		end = start + limit
	}

	return items[start:end]
}
