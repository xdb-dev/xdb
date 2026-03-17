package xdbsqlite

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// RecordKVTx handles record operations using KV-strategy tables.
type RecordKVTx struct {
	q   *xsql.Queries
	def *schema.Def
}

// ensureTable creates the KV table and registers a minimal flexible schema
// if neither exists. This is needed for the no-schema path where no
// CreateSchema call precedes record writes.
func (r *RecordKVTx) ensureTable(ctx context.Context, uri *core.URI) error {
	if r.def != nil {
		return nil // schema already exists, table was created via CreateSchema
	}

	table := kvTableName(uri)
	if err := r.q.CreateKVTable(ctx, xsql.CreateKVTableParams{Table: table}); err != nil {
		return err
	}

	// Register a minimal flexible schema so namespace/schema listing works.
	ns := uri.NS().String()
	sc := uri.Schema().String()

	exists, err := r.q.SchemaExists(ctx, xsql.SchemaExistsParams{
		Namespace: ns,
		Schema:    sc,
	})
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	data, err := json.Marshal(&schema.Def{
		URI:  core.New().NS(ns).Schema(sc).MustURI(),
		Mode: schema.ModeFlexible,
	})
	if err != nil {
		return err
	}

	return r.q.PutSchema(ctx, xsql.PutSchemaParams{
		Namespace: ns,
		Schema:    sc,
		Data:      data,
	})
}

func (r *RecordKVTx) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	if err := r.ensureTable(ctx, uri); err != nil {
		return nil, err
	}

	values, err := r.q.GetKVRecord(ctx, xsql.GetKVRecordParams{
		Table: kvTableName(uri),
		ID:    uri.ID().String(),
	})
	if err != nil {
		return nil, err
	}
	if values == nil {
		return nil, store.ErrNotFound
	}

	return kvRecordFromValues(uri, values), nil
}

func (r *RecordKVTx) ListRecords(ctx context.Context, uri *core.URI, q *store.ListQuery) (*store.Page[*core.Record], error) {
	if err := r.ensureTable(ctx, uri); err != nil {
		return nil, err
	}

	table := kvTableName(uri)
	total, err := r.q.CountKVRecords(ctx, xsql.CountKVRecordsParams{Table: table})
	if err != nil {
		return nil, err
	}

	limit, offset := paginationParams(q, total)

	rows, err := r.q.ListKVRecords(ctx, xsql.ListKVRecordsParams{
		Table:  table,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]*core.Record, len(rows))
	for i, row := range rows {
		rowURI := core.New().
			NS(uri.NS().String()).
			Schema(uri.Schema().String()).
			ID(row.ID).
			MustURI()
		items[i] = kvRecordFromValues(rowURI, row.Values)
	}

	return &store.Page[*core.Record]{
		Items:      items,
		Total:      total,
		NextOffset: nextOffset(offset, len(items), total),
	}, nil
}

func (r *RecordKVTx) CreateRecord(ctx context.Context, record *core.Record) error {
	if err := r.ensureTable(ctx, record.URI()); err != nil {
		return err
	}

	table := kvTableName(record.URI())
	exists, err := r.q.KVRecordExists(ctx, xsql.KVRecordExistsParams{
		Table: table,
		ID:    record.URI().ID().String(),
	})
	if err != nil {
		return err
	}
	if exists {
		return store.ErrAlreadyExists
	}

	return r.q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table:  table,
		ID:     record.URI().ID().String(),
		Values: kvValues(record),
	})
}

func (r *RecordKVTx) UpdateRecord(ctx context.Context, record *core.Record) error {
	if err := r.ensureTable(ctx, record.URI()); err != nil {
		return err
	}

	table := kvTableName(record.URI())
	exists, err := r.q.KVRecordExists(ctx, xsql.KVRecordExistsParams{
		Table: table,
		ID:    record.URI().ID().String(),
	})
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrNotFound
	}

	return r.q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table:  table,
		ID:     record.URI().ID().String(),
		Values: kvValues(record),
	})
}

func (r *RecordKVTx) UpsertRecord(ctx context.Context, record *core.Record) error {
	if err := r.ensureTable(ctx, record.URI()); err != nil {
		return err
	}

	return r.q.CreateKVRecord(ctx, xsql.CreateKVRecordParams{
		Table:  kvTableName(record.URI()),
		ID:     record.URI().ID().String(),
		Values: kvValues(record),
	})
}

func (r *RecordKVTx) DeleteRecord(ctx context.Context, uri *core.URI) error {
	if err := r.ensureTable(ctx, uri); err != nil {
		return err
	}

	table := kvTableName(uri)
	exists, err := r.q.KVRecordExists(ctx, xsql.KVRecordExistsParams{
		Table: table,
		ID:    uri.ID().String(),
	})
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrNotFound
	}

	return r.q.DeleteKVRecord(ctx, xsql.DeleteKVRecordParams{
		Table: table,
		ID:    uri.ID().String(),
	})
}

// RecordTableTx handles record operations using column-strategy tables.
type RecordTableTx struct {
	q   *xsql.Queries
	def *schema.Def
}

func (r *RecordTableTx) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	values, err := r.q.GetRecord(ctx, xsql.GetRecordParams{
		Table:   columnTableName(uri),
		ID:      uri.ID().String(),
		Columns: columnValues(r.def),
	})
	if err != nil {
		return nil, err
	}
	if values == nil {
		return nil, store.ErrNotFound
	}

	record := core.NewRecord(uri.NS().String(), uri.Schema().String(), uri.ID().String())
	for _, value := range values {
		record.Set(value.Name, value.Val)
	}
	return record, nil
}

func (r *RecordTableTx) ListRecords(ctx context.Context, uri *core.URI, q *store.ListQuery) (*store.Page[*core.Record], error) {
	table := columnTableName(uri)

	total, err := r.q.CountRecords(ctx, xsql.CountRecordsParams{Table: table})
	if err != nil {
		return nil, err
	}

	limit, offset := paginationParams(q, total)

	rows, err := r.q.ListRecords(ctx, xsql.ListRecordsParams{
		Table:   table,
		Columns: columnValues(r.def),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	ns := uri.NS().String()
	sc := uri.Schema().String()
	items := make([]*core.Record, len(rows))
	for i, row := range rows {
		// First value is _id.
		id := row[0].Val.Unwrap().(string)
		rec := core.NewRecord(ns, sc, id)
		for _, v := range row[1:] {
			rec.Set(v.Name, v.Val)
		}
		items[i] = rec
	}

	return &store.Page[*core.Record]{
		Items:      items,
		Total:      total,
		NextOffset: nextOffset(offset, len(items), total),
	}, nil
}

func (r *RecordTableTx) CreateRecord(ctx context.Context, record *core.Record) error {
	table := columnTableName(record.URI())

	exists, err := r.q.RecordExists(ctx, xsql.RecordExistsParams{
		Table: table,
		ID:    record.URI().ID().String(),
	})
	if err != nil {
		return err
	}
	if exists {
		return store.ErrAlreadyExists
	}

	return r.q.CreateRecord(ctx, xsql.CreateRecordParams{
		Table:  table,
		ID:     record.URI().ID().String(),
		Values: recordToValues(r.def, record),
	})
}

func (r *RecordTableTx) UpdateRecord(ctx context.Context, record *core.Record) error {
	table := columnTableName(record.URI())

	exists, err := r.q.RecordExists(ctx, xsql.RecordExistsParams{
		Table: table,
		ID:    record.URI().ID().String(),
	})
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrNotFound
	}

	return r.q.UpdateRecord(ctx, xsql.UpdateRecordParams{
		Table:  table,
		ID:     record.URI().ID().String(),
		Values: recordToValues(r.def, record),
	})
}

func (r *RecordTableTx) UpsertRecord(ctx context.Context, record *core.Record) error {
	return r.q.UpsertRecord(ctx, xsql.UpsertRecordParams{
		Table:  columnTableName(record.URI()),
		ID:     record.URI().ID().String(),
		Values: recordToValues(r.def, record),
	})
}

func (r *RecordTableTx) DeleteRecord(ctx context.Context, uri *core.URI) error {
	table := columnTableName(uri)

	exists, err := r.q.RecordExists(ctx, xsql.RecordExistsParams{
		Table: table,
		ID:    uri.ID().String(),
	})
	if err != nil {
		return err
	}
	if !exists {
		return store.ErrNotFound
	}

	return r.q.DeleteRecord(ctx, xsql.DeleteRecordParams{
		Table: table,
		ID:    uri.ID().String(),
	})
}

// --- Pagination helpers ---

// paginationParams extracts limit/offset from a ListQuery, defaulting to fetch all.
func paginationParams(q *store.ListQuery, total int) (limit, offset int) {
	if q == nil {
		return total, 0
	}
	limit = q.Limit
	if limit <= 0 {
		limit = total
	}
	return limit, q.Offset
}

// nextOffset computes the next page offset, returning 0 when there are no more pages.
func nextOffset(offset, fetched, total int) int {
	next := offset + fetched
	if next >= total {
		return 0
	}
	return next
}

// --- Strategy resolution ---

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
			q:   q,
			def: def,
		}, def, nil
	}
	return &RecordKVTx{
		q:   q,
		def: def,
	}, def, nil
}

// --- Store delegation ---

// GetRecord retrieves a single record by URI.
func (s *Store) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

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
// When uri has no schema component, lists records across all schemas in the namespace.
func (s *Store) ListRecords(
	ctx context.Context,
	uri *core.URI,
	lq *store.ListQuery,
) (*store.Page[*core.Record], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	q := xsql.NewQueries(tx)

	var res *store.Page[*core.Record]
	if uri.Schema() == nil {
		res, err = s.listRecordsByNamespace(ctx, q, uri, lq)
	} else {
		var rs store.RecordStore
		rs, _, err = s.recordStore(ctx, q, uri)
		if err == nil {
			res, err = rs.ListRecords(ctx, uri, lq)
		}
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}

// listRecordsByNamespace aggregates records across all schemas in a namespace.
func (s *Store) listRecordsByNamespace(
	ctx context.Context,
	q *xsql.Queries,
	uri *core.URI,
	lq *store.ListQuery,
) (*store.Page[*core.Record], error) {
	ns := uri.NS().String()
	schemas, err := q.ListSchemas(ctx, xsql.ListSchemasParams{
		Namespace: &ns,
		Limit:     10000,
	})
	if err != nil {
		return nil, err
	}

	var all []*core.Record
	for _, sc := range schemas {
		schemaURI := core.New().NS(ns).Schema(sc.Schema).MustURI()
		rs, _, err := s.recordStore(ctx, q, schemaURI)
		if err != nil {
			return nil, err
		}

		page, err := rs.ListRecords(ctx, schemaURI, nil)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].URI().Path() < all[j].URI().Path()
	})

	return store.Paginate(all, lq), nil
}

// CreateRecord creates a new record.
func (s *Store) CreateRecord(ctx context.Context, record *core.Record) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

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
	defer tx.Rollback() //nolint:errcheck

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
	defer tx.Rollback() //nolint:errcheck

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
	defer tx.Rollback() //nolint:errcheck

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
