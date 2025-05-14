package xdbsqlite

import (
	"context"
	"database/sql"
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
func (s *SQLStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, error) {
	return nil, nil
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
