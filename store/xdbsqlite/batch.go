package xdbsqlite

import (
	"context"
	"database/sql"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// ExecuteBatch runs fn within a SQLite transaction.
// If fn returns an error, all changes are rolled back.
func (s *Store) ExecuteBatch(ctx context.Context, fn func(tx store.Store) error) error {
	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer sqlTx.Rollback()

	q := xsql.NewQueries(sqlTx)
	tv := &storeTx{
		SchemaTx:    &SchemaTx{q: q},
		NamespaceTx: &NamespaceTx{q: q},
		store:       s,
		q:           q,
		tx:          sqlTx,
	}

	if err := fn(tv); err != nil {
		return err
	}

	return sqlTx.Commit()
}

// storeTx implements [store.Store] by embedding Tx types for schema/namespace
// and resolving record strategy dynamically per call.
type storeTx struct {
	*SchemaTx
	*NamespaceTx
	store *Store
	q     *xsql.Queries
	tx    *sql.Tx
}

func (t *storeTx) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	rs, _, err := t.store.recordStore(ctx, t.q, uri)
	if err != nil {
		return nil, err
	}
	return rs.GetRecord(ctx, uri)
}

func (t *storeTx) ListRecords(ctx context.Context, q *store.Query) (*store.Page[*core.Record], error) {
	uri := q.URI
	rs, _, err := t.store.recordStore(ctx, t.q, uri)
	if err != nil {
		return nil, err
	}
	return rs.ListRecords(ctx, q)
}

func (t *storeTx) CreateRecord(ctx context.Context, record *core.Record) error {
	rs, def, err := t.store.recordStore(ctx, t.q, record.URI())
	if err != nil {
		return err
	}
	if err := t.store.validateAndEvolve(ctx, t.q, def, record.Tuples()); err != nil {
		return err
	}
	return rs.CreateRecord(ctx, record)
}

func (t *storeTx) UpdateRecord(ctx context.Context, record *core.Record) error {
	rs, def, err := t.store.recordStore(ctx, t.q, record.URI())
	if err != nil {
		return err
	}
	if err := t.store.validateAndEvolve(ctx, t.q, def, record.Tuples()); err != nil {
		return err
	}
	return rs.UpdateRecord(ctx, record)
}

func (t *storeTx) UpsertRecord(ctx context.Context, record *core.Record) error {
	rs, def, err := t.store.recordStore(ctx, t.q, record.URI())
	if err != nil {
		return err
	}
	if err := t.store.validateAndEvolve(ctx, t.q, def, record.Tuples()); err != nil {
		return err
	}
	return rs.UpsertRecord(ctx, record)
}

func (t *storeTx) DeleteRecord(ctx context.Context, uri *core.URI) error {
	rs, _, err := t.store.recordStore(ctx, t.q, uri)
	if err != nil {
		return err
	}
	return rs.DeleteRecord(ctx, uri)
}

func (t *storeTx) Close() error { return nil }
