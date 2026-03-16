package xdbsqlite

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// SchemaTx holds a Queries instance for schema operations.
type SchemaTx struct {
	q *xsql.Queries
}

func (s *SchemaTx) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	return nil, store.ErrNotFound
}

func (s *SchemaTx) ListSchemas(ctx context.Context, uri *core.URI, q *store.ListQuery) (*store.Page[*schema.Def], error) {
	return nil, nil
}

func (s *SchemaTx) CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	return nil
}

func (s *SchemaTx) UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	return nil
}

func (s *SchemaTx) DeleteSchema(ctx context.Context, uri *core.URI) error {
	return nil
}

// --- Store delegation ---

// GetSchema retrieves a schema definition by URI.
func (s *Store) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	res, err := stx.GetSchema(ctx, uri)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// ListSchemas lists schemas, optionally scoped by namespace URI.
func (s *Store) ListSchemas(
	ctx context.Context,
	uri *core.URI,
	q *store.ListQuery,
) (*store.Page[*schema.Def], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	res, err := stx.ListSchemas(ctx, uri, q)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// CreateSchema creates a new schema definition.
func (s *Store) CreateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	if err := stx.CreateSchema(ctx, uri, def); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.cache.Store(cacheKey(uri), def)
	return nil
}

// UpdateSchema updates an existing schema definition. Mode changes are not allowed.
func (s *Store) UpdateSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	if err := stx.UpdateSchema(ctx, uri, def); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.cache.Store(cacheKey(uri), def)
	return nil
}

// DeleteSchema deletes a schema by URI.
func (s *Store) DeleteSchema(ctx context.Context, uri *core.URI) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stx := &SchemaTx{q: xsql.NewQueries(tx)}

	if err := stx.DeleteSchema(ctx, uri); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.invalidateSchema(uri)
	return nil
}
