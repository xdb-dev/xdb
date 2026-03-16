package xdbsqlite

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// RecordKVTx handles record operations using KV-strategy tables.
type RecordKVTx struct {
	q     *xsql.Queries
	def   *schema.Def
	table string
	id    string
}

func (r *RecordKVTx) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	return nil, store.ErrNotFound
}

func (r *RecordKVTx) ListRecords(ctx context.Context, uri *core.URI, q *store.ListQuery) (*store.Page[*core.Record], error) {
	return nil, nil
}

func (r *RecordKVTx) CreateRecord(ctx context.Context, record *core.Record) error {
	return nil
}

func (r *RecordKVTx) UpdateRecord(ctx context.Context, record *core.Record) error {
	return nil
}

func (r *RecordKVTx) UpsertRecord(ctx context.Context, record *core.Record) error {
	return nil
}

func (r *RecordKVTx) DeleteRecord(ctx context.Context, uri *core.URI) error {
	return nil
}

// RecordTableTx handles record operations using column-strategy tables.
type RecordTableTx struct {
	q     *xsql.Queries
	def   *schema.Def
	table string
	id    string
}

func (r *RecordTableTx) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	return nil, store.ErrNotFound
}

func (r *RecordTableTx) ListRecords(ctx context.Context, uri *core.URI, q *store.ListQuery) (*store.Page[*core.Record], error) {
	return nil, nil
}

func (r *RecordTableTx) CreateRecord(ctx context.Context, record *core.Record) error {
	return nil
}

func (r *RecordTableTx) UpdateRecord(ctx context.Context, record *core.Record) error {
	return nil
}

func (r *RecordTableTx) UpsertRecord(ctx context.Context, record *core.Record) error {
	return nil
}

func (r *RecordTableTx) DeleteRecord(ctx context.Context, uri *core.URI) error {
	return nil
}

// recordStore resolves the strategy and returns the appropriate Tx type.
func (s *Store) recordStore(
	ctx context.Context,
	q *xsql.Queries,
	uri *core.URI,
) (store.RecordStore, *schema.Def, error) {
	def, useTable, err := s.resolveStrategy(ctx, q, uri)
	if err != nil {
		return nil, nil, err
	}

	if useTable {
		return &RecordTableTx{
			q:     q,
			def:   def,
			table: columnTableName(uri),
			id:    uri.ID().String(),
		}, def, nil
	}
	return &RecordKVTx{
		q:     q,
		def:   def,
		table: kvTableName(uri),
		id:    uri.ID().String(),
	}, def, nil
}

// --- Store delegation ---

// GetRecord retrieves a single record by URI.
func (s *Store) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := xsql.NewQueries(tx)
	rs, _, err := s.recordStore(ctx, q, uri)
	if err != nil {
		return nil, err
	}

	res, err := rs.GetRecord(ctx, uri)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// ListRecords lists records matching the given query.
func (s *Store) ListRecords(
	ctx context.Context,
	uri *core.URI,
	lq *store.ListQuery,
) (*store.Page[*core.Record], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := xsql.NewQueries(tx)
	rs, _, err := s.recordStore(ctx, q, uri)
	if err != nil {
		return nil, err
	}

	res, err := rs.ListRecords(ctx, uri, lq)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// CreateRecord creates a new record.
func (s *Store) CreateRecord(ctx context.Context, record *core.Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := xsql.NewQueries(tx)
	rs, def, err := s.recordStore(ctx, q, record.URI())
	if err != nil {
		return err
	}

	if err := s.validateAndEvolve(ctx, q, def, record.Tuples()); err != nil {
		return err
	}

	if err := rs.CreateRecord(ctx, record); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateRecord updates an existing record.
func (s *Store) UpdateRecord(ctx context.Context, record *core.Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := xsql.NewQueries(tx)
	rs, def, err := s.recordStore(ctx, q, record.URI())
	if err != nil {
		return err
	}

	if err := s.validateAndEvolve(ctx, q, def, record.Tuples()); err != nil {
		return err
	}

	if err := rs.UpdateRecord(ctx, record); err != nil {
		return err
	}

	return tx.Commit()
}

// UpsertRecord creates or updates a record.
func (s *Store) UpsertRecord(ctx context.Context, record *core.Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := xsql.NewQueries(tx)
	rs, def, err := s.recordStore(ctx, q, record.URI())
	if err != nil {
		return err
	}

	if err := s.validateAndEvolve(ctx, q, def, record.Tuples()); err != nil {
		return err
	}

	if err := rs.UpsertRecord(ctx, record); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteRecord deletes a record by URI.
func (s *Store) DeleteRecord(ctx context.Context, uri *core.URI) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := xsql.NewQueries(tx)
	rs, _, err := s.recordStore(ctx, q, uri)
	if err != nil {
		return err
	}

	if err := rs.DeleteRecord(ctx, uri); err != nil {
		return err
	}

	return tx.Commit()
}
