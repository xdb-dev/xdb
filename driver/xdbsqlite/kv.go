// Package xdbsqlite provides a SQLite-backed driver for XDB.
package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
func (kv *KVStore) GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error) {
	if len(uris) == 0 {
		return nil, nil, nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	tuplesMap := make(map[string]*core.Tuple)

	encodedKeys := x.Map(uris, func(uri *core.URI) string {
		id, attr, err := kv.codec.EncodeURI(uri)
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

		xu, err := kv.codec.DecodeURI([]byte(id), []byte(attr))
		if err != nil {
			return nil, nil, err
		}

		xv, err := kv.codec.DecodeValue(value)
		if err != nil {
			return nil, nil, err
		}

		tuple := core.NewTuple(xu.Repo(), xu.ID(), xu.Attr(), xv)
		tuplesMap[xu.String()] = tuple
	}

	tuples := make([]*core.Tuple, 0, len(uris))
	missing := make([]*core.URI, 0, len(uris))

	for _, uri := range uris {
		tuple, ok := tuplesMap[uri.String()]
		if !ok {
			missing = append(missing, uri)
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
		id, attr, err := kv.codec.EncodeURI(tuple.URI())
		if err != nil {
			return err
		}

		v, err := kv.codec.EncodeValue(tuple.Value())
		if err != nil {
			return err
		}

		insertRecord := goqu.Record{
			"key":        string(id) + ":" + string(attr),
			"id":         string(id),
			"attr":       string(attr),
			"value":      v,
			"updated_at": time.Now().UnixMilli(),
		}
		putRecords = append(putRecords, insertRecord)
	}

	insertQuery := goqu.Insert("xdbkvstore").
		Prepared(true).
		Rows(putRecords).
		OnConflict(goqu.DoUpdate("key", goqu.Record{
			"value":      goqu.I("EXCLUDED.value"),
			"updated_at": goqu.I("EXCLUDED.updated_at"),
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
func (kv *KVStore) DeleteTuples(ctx context.Context, uris []*core.URI) error {
	if len(uris) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	encodedKeys := x.Map(uris, func(uri *core.URI) string {
		id, attr, err := kv.codec.EncodeURI(uri)
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
func (kv *KVStore) GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error) {
	if len(uris) == 0 {
		return nil, nil, nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	encodedIDs := x.Map(uris, func(uri *core.URI) string {
		id, err := kv.codec.EncodeID(uri.ID())
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
	repoMap := make(map[string]string)

	for rows.Next() {
		var id string
		var attr string
		var value []byte
		err := rows.Scan(&id, &attr, &value)
		if err != nil {
			return nil, nil, err
		}

		xu, err := kv.codec.DecodeURI([]byte(id), []byte(attr))
		if err != nil {
			return nil, nil, err
		}

		xv, err := kv.codec.DecodeValue(value)
		if err != nil {
			return nil, nil, err
		}

		idStr := xu.ID().String()
		_, ok := recordsMap[idStr]
		if !ok {
			recordsMap[idStr] = core.NewRecord(xu.Repo(), xu.ID()...)
			repoMap[idStr] = xu.Repo()
		}

		recordsMap[idStr].Set(xu.Attr(), xv)
	}

	records := make([]*core.Record, 0, len(recordsMap))
	missing := make([]*core.URI, 0, len(uris))

	for _, uri := range uris {
		record, ok := recordsMap[uri.ID().String()]
		if !ok {
			missing = append(missing, uri)
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
func (kv *KVStore) DeleteRecords(ctx context.Context, uris []*core.URI) error {
	if len(uris) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	encodedIDs := x.Map(uris, func(uri *core.URI) string {
		id, err := kv.codec.EncodeID(uri.ID())
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
				value BLOB,
				updated_at INTEGER
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
