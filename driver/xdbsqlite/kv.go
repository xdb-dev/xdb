// Package xdbsqlite provides a SQLite-backed driver for XDB.
package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/codec/json"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/x"
)

var (
	_ driver.TupleReader  = (*KVStore)(nil)
	_ driver.TupleWriter  = (*KVStore)(nil)
	_ driver.RecordReader = (*KVStore)(nil)
	_ driver.RecordWriter = (*KVStore)(nil)
)

// KVStore is a key-value store for SQLite.
// It stores tuples in SQLite tables as key-value pairs.
type KVStore struct {
	db    *sql.DB
	codec codec.KVCodec
}

// NewKVStore creates a new SQLite KVStore.
func NewKVStore(db *sql.DB) *KVStore {
	return &KVStore{db: db, codec: json.New()}
}

// GetTuples gets tuples from the SQLite key-value table.
func (kv *KVStore) GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	tuplesMap := make(map[string]*core.Tuple)

	encodedKeys := x.Map(keys, func(key *core.Key) string {
		id, attr, err := kv.codec.EncodeKey(key)
		if err != nil {
			return ""
		}
		return string(id) + ":" + string(attr)
	})

	getQuery := goqu.Select("id", "attr", "value").
		From("xdbkvstore").
		Where(goqu.Ex{
			"key": encodedKeys,
		})

	query, args, err := getQuery.ToSQL()
	if err != nil {
		return nil, nil, err
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}

	for rows.Next() {
		var id string
		var attr string
		var value []byte
		err := rows.Scan(&id, &attr, &value)
		if err != nil {
			return nil, nil, err
		}

		xk, err := kv.codec.DecodeKey([]byte(id), []byte(attr))
		if err != nil {
			return nil, nil, err
		}

		xv, err := kv.codec.DecodeValue(value)
		if err != nil {
			return nil, nil, err
		}

		tuple := xk.Value(xv)
		tuplesMap[xk.String()] = tuple
	}

	tuples := make([]*core.Tuple, 0, len(keys))
	missing := make([]*core.Key, 0, len(keys))

	for _, key := range keys {
		tuple, ok := tuplesMap[key.String()]
		if !ok {
			missing = append(missing, key)
			continue
		}

		tuples = append(tuples, tuple)
	}

	return tuples, missing, nil

}

// PutTuples puts tuples into the key-value store.
func (kv *KVStore) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	putRecords := make([]goqu.Record, 0, len(tuples))

	for _, tuple := range tuples {
		id, attr, err := kv.codec.EncodeKey(tuple.Key())
		if err != nil {
			return err
		}

		v, err := kv.codec.EncodeValue(tuple.Value())
		if err != nil {
			return err
		}

		insertRecord := goqu.Record{
			"key":   string(id) + ":" + string(attr),
			"id":    string(id),
			"attr":  string(attr),
			"value": v,
		}
		putRecords = append(putRecords, insertRecord)
	}

	insertQuery := goqu.Insert("xdbkvstore").
		Prepared(true).
		Rows(putRecords).
		OnConflict(goqu.DoUpdate("key", goqu.Record{
			"value": goqu.I("EXCLUDED.value"),
		}))

	query, args, err := insertQuery.ToSQL()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteTuples deletes tuples from the key-value store.
func (kv *KVStore) DeleteTuples(ctx context.Context, keys []*core.Key) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	encodedKeys := x.Map(keys, func(key *core.Key) string {
		id, attr, err := kv.codec.EncodeKey(key)
		if err != nil {
			return ""
		}
		return string(id) + ":" + string(attr)
	})

	deleteQuery := goqu.Delete("xdbkvstore").
		Where(goqu.Ex{
			"key": encodedKeys,
		})

	query, args, err := deleteQuery.ToSQL()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetRecords gets records from the key-value store.
func (kv *KVStore) GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	encodedIDs := x.Map(keys, func(key *core.Key) string {
		id, err := kv.codec.EncodeID(key.ID())
		if err != nil {
			return ""
		}
		return string(id)
	})

	selectQuery := goqu.Select("id", "attr", "value").
		From("xdbkvstore").
		Where(goqu.Ex{
			"id": encodedIDs,
		})

	query, args, err := selectQuery.ToSQL()
	if err != nil {
		return nil, nil, err
	}

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}

	recordsMap := make(map[string]*core.Record)

	for rows.Next() {
		var id string
		var attr string
		var value []byte
		err := rows.Scan(&id, &attr, &value)
		if err != nil {
			return nil, nil, err
		}

		xk, err := kv.codec.DecodeKey([]byte(id), []byte(attr))
		if err != nil {
			return nil, nil, err
		}

		xv, err := kv.codec.DecodeValue(value)
		if err != nil {
			return nil, nil, err
		}

		_, ok := recordsMap[xk.ID().String()]
		if !ok {
			recordsMap[xk.ID().String()] = core.NewRecord(xk.ID()...)
		}

		recordsMap[xk.ID().String()].Set(xk.Attr(), xv)
	}

	records := make([]*core.Record, 0, len(recordsMap))
	missing := make([]*core.Key, 0, len(keys))

	for _, key := range keys {
		record, ok := recordsMap[key.ID().String()]
		if !ok {
			missing = append(missing, key)
			continue
		}

		records = append(records, record)
	}

	return records, missing, nil
}

// PutRecords puts records into the key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*core.Record) error {
	tuples := make([]*core.Tuple, 0, len(records))

	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}

	return kv.PutTuples(ctx, tuples)
}

// DeleteRecords deletes records from the key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*core.Key) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	encodedIDs := x.Map(keys, func(key *core.Key) string {
		id, err := kv.codec.EncodeID(key.ID())
		if err != nil {
			return ""
		}
		return string(id)
	})

	deleteQuery := goqu.Delete("xdbkvstore").
		Where(goqu.Ex{
			"id": encodedIDs,
		})

	query, args, err := deleteQuery.ToSQL()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Migrate creates tables for all provided kinds with the required schema and indexes.
func (kv *KVStore) Migrate(ctx context.Context) error {
	name := "xdbkvstore"
	tableStmt := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key TEXT PRIMARY KEY,
				id TEXT,
				attr TEXT,
				value BLOB
			);
		`, name)
	_, err := kv.db.ExecContext(ctx, tableStmt)
	if err != nil {
		return err
	}

	idxID := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_id ON %s (id);", name, name)
	_, err = kv.db.ExecContext(ctx, idxID)
	if err != nil {
		return err
	}

	idxAttr := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_attr ON %s (attr);", name, name)
	_, err = kv.db.ExecContext(ctx, idxAttr)
	if err != nil {
		return err
	}

	return nil
}
