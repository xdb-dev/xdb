package xdbsqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

var (
	_ driver.TupleReader  = (*SQLStore)(nil)
	_ driver.TupleWriter  = (*SQLStore)(nil)
	_ driver.RecordReader = (*SQLStore)(nil)
	_ driver.RecordWriter = (*SQLStore)(nil)
)

// SQLStore is a store that uses SQLite as the underlying database.
type SQLStore struct {
	db *sql.DB
}

// New creates a new SQLite driver
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

// GetTuples gets tuples from the SQLite database.
func (s *SQLStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	grouped := x.GroupAttrs(keys...)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	defer tx.Rollback()

	for kind, rows := range grouped {
		for id, attrs := range rows {
			query, args, err := goqu.Select(attrs).
				From(kind).
				Where(goqu.L("id = ?", id)).
				ToSQL()

			rows, err := tx.QueryContext(ctx, query, args...)
			if err != nil {
				return nil, nil, err
			}

			defer rows.Close()
		}
	}
	return nil, nil, nil
}

// PutTuples puts tuples into the SQLite database.
func (s *SQLStore) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
	grouped := x.GroupTuples(tuples...)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	for kind, rows := range grouped {
		for id, tuples := range rows {
			insertRecord := goqu.Record{"id": id}
			updateRecord := goqu.Record{}

			for _, tuple := range tuples {
				insertRecord[tuple.Attr()] = tuple.Value().Unwrap()
				updateRecord[tuple.Attr()] = goqu.I("EXCLUDED." + tuple.Attr())
			}

			insertQuery := goqu.Insert(kind).
				Prepared(true).
				Rows(insertRecord).
				OnConflict(goqu.DoUpdate("id", updateRecord))

			query, args, err := insertQuery.ToSQL()
			if err != nil {
				return err
			}

			fmt.Println(query)
			fmt.Println(args)

			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// DeleteTuples deletes tuples from the SQLite database.
func (s *SQLStore) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	return nil
}

// GetRecords gets records from the SQLite database.
func (s *SQLStore) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	return nil, nil, nil
}

// PutRecords puts records into the SQLite database.
func (s *SQLStore) PutRecords(ctx context.Context, records []*types.Record) error {
	return nil
}

// DeleteRecords deletes records from the SQLite database.
func (s *SQLStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	return nil
}

// encodeValue encodes a value to SQLite compatible type.
// mapping:
// - string -> TEXT
// - int -> INTEGER
// - float -> REAL
// - bool -> INTEGER
// - bytes -> BLOB
// - time -> INTEGER
// - point -> TEXT(JSON)
// - []string -> TEXT(JSON)
// - []int -> TEXT(JSON)
// - []float -> TEXT(JSON)
// - []bool -> TEXT(JSON)
// - []bytes -> TEXT(JSON)
// - []time -> TEXT(JSON)
// - []point -> TEXT(JSON)
func encodeValue(value *types.Value) (any, error) {
	if value.Repeated() {
		// encode arrays as JSON
		return json.Marshal(value.Unwrap())
	}

	switch value.TypeID() {
	case types.TypeString:
		return value.ToString(), nil
	case types.TypeInteger:
		return value.ToInt(), nil
	case types.TypeFloat:
		return value.ToFloat(), nil
	case types.TypeBoolean:
		return value.ToBool(), nil
	case types.TypeBytes:
		return value.ToBytes(), nil
	case types.TypeTime:
		return value.ToTime(), nil
	case types.TypePoint:
		return value.ToPoint(), nil
	}

	return nil, fmt.Errorf("unsupported value type: %s", value.TypeID())
}

func decodeValue(value any) (*types.Value, error) {
	return nil, nil
}
