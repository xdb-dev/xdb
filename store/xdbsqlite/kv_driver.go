package xdbsqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/xdb-dev/xdb/codec/json"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store/xdbsqlite/internal"
)

// KVDriverTx implements KV-based record operations for flexible mode schemas.
type KVDriverTx struct {
	queries   *internal.Queries
	jsonCodec *json.Codec
	schemaDef *schema.Def
}

// NewKVDriverTx creates a new KVDriverTx with the given transaction and schema definition.
func NewKVDriverTx(tx *sql.Tx, schemaDef *schema.Def) *KVDriverTx {
	return &KVDriverTx{
		queries:   internal.NewQueries(tx),
		jsonCodec: json.New(),
		schemaDef: schemaDef,
	}
}

func (d *KVDriverTx) tableName() string {
	return normalize(d.schemaDef.NS.String() + "__" + d.schemaDef.Name)
}

// GetRecords retrieves records by IDs from the KV table.
func (d *KVDriverTx) GetRecords(
	ctx context.Context,
	ids []string,
) ([]*core.Record, []*core.URI, error) {
	rows, err := d.queries.SelectKVTuples(ctx, internal.SelectKVTuplesParams{
		Table: d.tableName(),
		IDs:   ids,
	})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	schemaURI := d.schemaDef.URI()

	recordMap, err := d.scanKVRows(rows)
	if err != nil {
		return nil, nil, err
	}

	records := make([]*core.Record, 0, len(recordMap))
	var missed []*core.URI
	for _, id := range ids {
		if record, ok := recordMap[id]; ok {
			records = append(records, record)
		} else {
			missed = append(missed, core.New().
				NS(schemaURI.NS().String()).
				Schema(schemaURI.Schema().String()).
				ID(id).
				MustURI())
		}
	}

	return records, missed, nil
}

func (d *KVDriverTx) scanKVRows(
	rows *sql.Rows,
) (map[string]*core.Record, error) {
	schemaURI := d.schemaDef.URI()
	recordMap := make(map[string]*core.Record)

	for rows.Next() {
		var kvTuple internal.KVTuple
		err := rows.Scan(&kvTuple.Key, &kvTuple.ID, &kvTuple.Attr, &kvTuple.Value, &kvTuple.UpdatedAt)
		if err != nil {
			return nil, err
		}

		record, ok := recordMap[kvTuple.ID]
		if !ok {
			record = core.NewRecord(schemaURI.NS().String(), schemaURI.Schema().String(), kvTuple.ID)
			recordMap[kvTuple.ID] = record
		}

		coreVal, err := d.jsonCodec.DecodeValue([]byte(kvTuple.Value))
		if err != nil {
			return nil, err
		}

		if coreVal != nil {
			record.Set(kvTuple.Attr, coreVal)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return recordMap, nil
}

// GetTuples retrieves individual tuples by their keys from the KV table.
func (d *KVDriverTx) GetTuples(
	ctx context.Context,
	keys []string,
) ([]*core.Tuple, []*core.URI, error) {
	schemaURI := d.schemaDef.URI()
	rows, err := d.queries.SelectKVTuplesByKeys(ctx, internal.SelectKVTuplesByKeysParams{
		Table: d.tableName(),
		Keys:  keys,
	})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	found := make(map[string]*core.Tuple)
	for rows.Next() {
		var kvTuple internal.KVTuple
		err := rows.Scan(&kvTuple.Key, &kvTuple.ID, &kvTuple.Attr, &kvTuple.Value, &kvTuple.UpdatedAt)
		if err != nil {
			return nil, nil, err
		}

		coreVal, err := d.jsonCodec.DecodeValue([]byte(kvTuple.Value))
		if err != nil {
			return nil, nil, err
		}

		path := schemaURI.NS().String() + "/" + schemaURI.Schema().String() + "/" + kvTuple.ID
		found[kvTuple.Key] = core.NewTuple(path, kvTuple.Attr, coreVal)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var tuples []*core.Tuple
	var missed []*core.URI
	for _, key := range keys {
		if t, ok := found[key]; ok {
			tuples = append(tuples, t)
		} else {
			parts := splitKey(key)
			if parts != nil {
				missed = append(missed, core.New().
					NS(schemaURI.NS().String()).
					Schema(schemaURI.Schema().String()).
					ID(parts[0]).
					Attr(parts[1]).
					MustURI())
			}
		}
	}

	return tuples, missed, nil
}

// PutTuples saves individual tuples to the KV table.
func (d *KVDriverTx) PutTuples(
	ctx context.Context,
	tuples []*core.Tuple,
) error {
	tbl := d.tableName()
	now := time.Now().Unix()

	kvTuples := make([]internal.KVTuple, 0, len(tuples))
	for _, tuple := range tuples {
		id := tuple.ID().String()
		attr := tuple.Attr().String()

		encodedValue, err := d.jsonCodec.EncodeValue(tuple.Value())
		if err != nil {
			return err
		}

		kvTuples = append(kvTuples, internal.KVTuple{
			Key:       id + "#" + attr,
			ID:        id,
			Attr:      attr,
			Value:     string(encodedValue),
			UpdatedAt: now,
		})
	}

	return d.queries.InsertKVTuples(ctx, internal.InsertKVTuplesParams{
		Table:  tbl,
		Tuples: kvTuples,
	})
}

// DeleteTuples removes individual tuples by their keys from the KV table.
func (d *KVDriverTx) DeleteTuples(ctx context.Context, keys []string) error {
	return d.queries.DeleteKVTuplesByKeys(ctx, internal.DeleteKVTuplesByKeysParams{
		Table: d.tableName(),
		Keys:  keys,
	})
}

func splitKey(key string) []string {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '#' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return nil
}

// PutRecords saves records to the KV table.
func (d *KVDriverTx) PutRecords(
	ctx context.Context,
	records []*core.Record,
) error {
	tbl := d.tableName()
	now := time.Now().Unix()

	var kvTuples []internal.KVTuple

	for _, record := range records {
		id := record.ID().String()

		for _, tuple := range record.Tuples() {
			attr := tuple.Attr().String()

			encodedValue, err := d.jsonCodec.EncodeValue(tuple.Value())
			if err != nil {
				return err
			}

			kvTuples = append(kvTuples, internal.KVTuple{
				Key:       id + "#" + attr,
				ID:        id,
				Attr:      attr,
				Value:     string(encodedValue),
				UpdatedAt: now,
			})
		}
	}

	return d.queries.InsertKVTuples(ctx, internal.InsertKVTuplesParams{
		Table:  tbl,
		Tuples: kvTuples,
	})
}

// DeleteRecords removes records from the KV table.
func (d *KVDriverTx) DeleteRecords(ctx context.Context, ids []string) error {
	tbl := d.tableName()
	return d.queries.DeleteKVTuples(ctx, internal.DeleteKVTuplesParams{
		Table: tbl,
		IDs:   ids,
	})
}
