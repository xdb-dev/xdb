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
	schemaURI *core.URI,
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

	recordMap, err := d.scanKVRows(rows, schemaURI)
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
	schemaURI *core.URI,
) (map[string]*core.Record, error) {
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
