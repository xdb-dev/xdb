package xdbredis

import (
	"context"
	"fmt"
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/store"
)

// GetRecord retrieves a single record by URI.
func (s *Store) GetRecord(ctx context.Context, uri *core.URI) (*core.Record, error) {
	key := s.recordKey(uri)

	fields, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: hgetall: %w", err)
	}
	if len(fields) == 0 {
		return nil, store.ErrNotFound
	}

	return decodeRecord(uri, fields)
}

// ListRecords lists records scoped by the given URI.
func (s *Store) ListRecords(
	ctx context.Context,
	q *store.Query,
) (*store.Page[*core.Record], error) {
	uri := q.URI

	if uri.Schema() != nil {
		return s.listRecordsBySchema(ctx, q)
	}

	// List across all schemas in the namespace.
	schemaNames, err := s.client.SMembers(ctx, s.schemaIndexKey(uri)).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: list schema index: %w", err)
	}

	var records []*core.Record
	for _, name := range schemaNames {
		schemaURI := core.New().NS(uri.NS().String()).Schema(name).MustURI()
		recs, err := s.fetchRecordsBySchema(ctx, schemaURI)
		if err != nil {
			return nil, err
		}
		records = append(records, recs...)
	}

	if q.Filter != "" {
		f, err := filter.Compile(q.Filter, nil)
		if err != nil {
			return nil, err
		}
		records, err = filter.Records(f, records)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].URI().Path() < records[j].URI().Path()
	})

	return store.Paginate(records, q), nil
}

// listRecordsBySchema lists records for a single schema.
func (s *Store) listRecordsBySchema(
	ctx context.Context,
	q *store.Query,
) (*store.Page[*core.Record], error) {
	uri := q.URI

	records, err := s.fetchRecordsBySchema(ctx, uri)
	if err != nil {
		return nil, err
	}

	if q.Filter != "" {
		f, err := filter.Compile(q.Filter, nil)
		if err != nil {
			return nil, err
		}
		records, err = filter.Records(f, records)
		if err != nil {
			return nil, err
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].URI().Path() < records[j].URI().Path()
	})

	return store.Paginate(records, q), nil
}

// fetchRecordsBySchema fetches all records for a given namespace+schema URI.
func (s *Store) fetchRecordsBySchema(
	ctx context.Context,
	uri *core.URI,
) ([]*core.Record, error) {
	idxKey := s.recordIndexKey(uri)

	ids, err := s.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: list record index: %w", err)
	}

	ns := uri.NS().String()
	schemaName := uri.Schema().String()

	records := make([]*core.Record, 0, len(ids))
	for _, id := range ids {
		recURI := core.New().NS(ns).Schema(schemaName).ID(id).MustURI()

		fields, err := s.client.HGetAll(ctx, s.recordKey(recURI)).Result()
		if err != nil {
			return nil, fmt.Errorf("xdbredis: hgetall %s: %w", id, err)
		}
		if len(fields) == 0 {
			continue
		}

		record, err := decodeRecord(recURI, fields)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// CreateRecord creates a new record.
func (s *Store) CreateRecord(ctx context.Context, record *core.Record) error {
	key := s.recordKey(record.URI())

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check record exists: %w", err)
	}
	if exists > 0 {
		return store.ErrAlreadyExists
	}

	return s.writeRecord(ctx, record)
}

// UpdateRecord updates an existing record.
func (s *Store) UpdateRecord(ctx context.Context, record *core.Record) error {
	key := s.recordKey(record.URI())

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check record exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}

	return s.writeRecord(ctx, record)
}

// UpsertRecord creates or updates a record unconditionally.
func (s *Store) UpsertRecord(ctx context.Context, record *core.Record) error {
	return s.writeRecord(ctx, record)
}

// DeleteRecord deletes a record by URI.
func (s *Store) DeleteRecord(ctx context.Context, uri *core.URI) error {
	key := s.recordKey(uri)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check record exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}

	idxKey := s.recordIndexKey(uri)
	id := uri.ID().String()

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, idxKey, id)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("xdbredis: delete record: %w", err)
	}

	return nil
}

// sentinelField is a marker field so empty records (no tuples) can be stored.
const sentinelField = "_"

// writeRecord encodes and writes a record to Redis with index update.
func (s *Store) writeRecord(ctx context.Context, record *core.Record) error {
	key := s.recordKey(record.URI())
	idxKey := s.recordIndexKey(record.URI())
	id := record.ID().String()

	fields, err := encodeRecord(record)
	if err != nil {
		return err
	}

	// Always include sentinel so HSet never receives an empty map.
	fields[sentinelField] = "1"

	schemaIdxKey := s.schemaIndexKey(record.URI())
	schemaName := record.Schema().String()

	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, fields)
	pipe.SAdd(ctx, idxKey, id)
	pipe.SAdd(ctx, schemaIdxKey, schemaName)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("xdbredis: write record: %w", err)
	}

	return nil
}

// encodeRecord converts a record's tuples into a map of field→encoded value.
func encodeRecord(record *core.Record) (map[string]any, error) {
	tuples := record.Tuples()
	fields := make(map[string]any, len(tuples))

	for _, t := range tuples {
		encoded, err := encodeValue(t.Value())
		if err != nil {
			return nil, fmt.Errorf(
				"xdbredis: encode field %s: %w",
				t.Attr().String(),
				err,
			)
		}
		fields[t.Attr().String()] = encoded
	}

	return fields, nil
}

// decodeRecord converts Redis hash fields into a [core.Record].
func decodeRecord(
	uri *core.URI,
	fields map[string]string,
) (*core.Record, error) {
	record := core.NewRecord(
		uri.NS().String(),
		uri.Schema().String(),
		uri.ID().String(),
	)

	for attr, encoded := range fields {
		if attr == sentinelField {
			continue
		}

		val, err := decodeValue(encoded)
		if err != nil {
			return nil, fmt.Errorf("xdbredis: decode field %s: %w", attr, err)
		}
		record.Set(attr, val)
	}

	return record, nil
}
