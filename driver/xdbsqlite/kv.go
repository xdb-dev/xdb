package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/encoding/xdbkv"
	"github.com/xdb-dev/xdb/types"
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
	db *sql.DB
}

// NewKVStore creates a new SQLite KVStore.
func NewKVStore(db *sql.DB) *KVStore {
	return &KVStore{db: db}
}

// GetTuples gets tuples from the SQLite key-value table.
func (kv *KVStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	grouped := x.GroupBy(keys, func(key *types.Key) string {
		return key.Kind()
	})

	tuplesMap := make(map[string]*types.Tuple)

	for kind, keys := range grouped {
		encodedKeys := x.Map(keys, func(key *types.Key) string {
			return string(xdbkv.EncodeKey(key))
		})

		getQuery := goqu.Select("key", "value").
			From(kind).
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
			var key string
			var value []byte
			err := rows.Scan(&key, &value)
			if err != nil {
				return nil, nil, err
			}

			xk, err := xdbkv.DecodeKey([]byte(key))
			if err != nil {
				return nil, nil, err
			}

			xv, err := xdbkv.DecodeValue(value)
			if err != nil {
				return nil, nil, err
			}

			tuple := types.NewTuple(
				xk.Kind(),
				xk.ID(),
				xk.Attr(),
				xv,
			)

			tuplesMap[tuple.Key().String()] = tuple
		}
	}

	tuples := make([]*types.Tuple, 0, len(keys))
	missing := make([]*types.Key, 0, len(keys))

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
func (kv *KVStore) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	grouped := x.GroupTuples(tuples...)

	for kind, rows := range grouped {
		putRecords := make([]goqu.Record, 0, len(rows))

		for _, tuples := range rows {
			for _, tuple := range tuples {
				k, v, err := xdbkv.EncodeTuple(tuple)
				if err != nil {
					return err
				}

				insertRecord := goqu.Record{
					"key":   string(k),
					"id":    tuple.ID(),
					"attr":  tuple.Attr(),
					"value": v,
				}

				putRecords = append(putRecords, insertRecord)
			}
		}

		insertQuery := goqu.Insert(kind).
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
	}

	return tx.Commit()
}

// DeleteTuples deletes tuples from the key-value store.
func (kv *KVStore) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	grouped := x.GroupBy(keys, func(key *types.Key) string {
		return key.Kind()
	})

	for kind, keys := range grouped {
		encodedKeys := x.Map(keys, func(key *types.Key) string {
			return string(xdbkv.EncodeKey(key))
		})

		deleteQuery := goqu.Delete(kind).
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
	}

	return tx.Commit()
}

// GetRecords gets records from the key-value store.
func (kv *KVStore) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	grouped := x.GroupBy(keys, func(key *types.Key) string {
		return key.Kind()
	})

	recordsMap := make(map[string]*types.Record)

	for kind, keys := range grouped {
		ids := x.Map(keys, func(key *types.Key) string {
			return key.ID()
		})

		selectQuery := goqu.Select("key", "value").
			From(kind).
			Where(goqu.Ex{
				"id": ids,
			})

		query, args, err := selectQuery.ToSQL()
		if err != nil {
			return nil, nil, err
		}

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, nil, err
		}

		for rows.Next() {
			var key string
			var value []byte
			err := rows.Scan(&key, &value)
			if err != nil {
				return nil, nil, err
			}

			xk, err := xdbkv.DecodeKey([]byte(key))
			if err != nil {
				return nil, nil, err
			}

			xv, err := xdbkv.DecodeValue(value)
			if err != nil {
				return nil, nil, err
			}

			_, ok := recordsMap[recordKey(xk)]
			if !ok {
				recordsMap[recordKey(xk)] = types.NewRecord(xk.Kind(), xk.ID())
			}

			recordsMap[recordKey(xk)].Set(xk.Attr(), xv)
		}
	}

	records := make([]*types.Record, 0, len(recordsMap))
	missing := make([]*types.Key, 0, len(keys))

	for _, key := range keys {
		record, ok := recordsMap[recordKey(key)]
		if !ok {
			missing = append(missing, key)
			continue
		}

		records = append(records, record)
	}

	return records, missing, nil
}

// PutRecords puts records into the key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*types.Record) error {
	tuples := make([]*types.Tuple, 0, len(records))

	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}

	return kv.PutTuples(ctx, tuples)
}

// DeleteRecords deletes records from the key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := kv.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	grouped := x.GroupBy(keys, func(key *types.Key) string {
		return key.Kind()
	})

	for kind, keys := range grouped {
		ids := x.Map(keys, func(key *types.Key) string {
			return key.ID()
		})

		deleteQuery := goqu.Delete(kind).
			Where(goqu.Ex{
				"id": ids,
			})

		query, args, err := deleteQuery.ToSQL()
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Migrate creates tables for all provided kinds with the required schema and indexes.
func (kv *KVStore) Migrate(ctx context.Context, kinds []string) error {
	for _, kind := range kinds {
		tableStmt := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				key TEXT PRIMARY KEY,
				id TEXT,
				attr TEXT,
				value BLOB
			);
		`, kind)
		_, err := kv.db.ExecContext(ctx, tableStmt)
		if err != nil {
			return err
		}

		idxID := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_id ON %s (id);", kind, kind)
		_, err = kv.db.ExecContext(ctx, idxID)
		if err != nil {
			return err
		}

		idxAttr := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_attr ON %s (attr);", kind, kind)
		_, err = kv.db.ExecContext(ctx, idxAttr)
		if err != nil {
			return err
		}
	}
	return nil
}

func recordKey[T interface {
	Kind() string
	ID() string
}](r T) string {
	return fmt.Sprintf("%s:%s", r.Kind(), r.ID())
}
