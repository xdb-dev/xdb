package sqlite

import (
	"context"
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/xdb-dev/xdb"
	"github.com/xdb-dev/xdb/schema"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type SQLiteStore struct {
	db     *sqlite.Conn
	schema *schema.Schema
}

func NewSQLiteStore(db *sqlite.Conn, schema *schema.Schema) *SQLiteStore {
	return &SQLiteStore{db: db, schema: schema}
}

func (s *SQLiteStore) PutRecord(ctx context.Context, record *xdb.Record) error {
	meta := s.schema.Get(record.Key().Kind())
	if meta == nil {
		return errors.New("kind not found")
	}

	query, args, err := updateRecordQuery(meta.Table, record)
	if err != nil {
		return err
	}

	err = sqlitex.Execute(s.db, query, &sqlitex.ExecOptions{Args: args})
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStore) GetRecord(ctx context.Context, key *xdb.Key) (*xdb.Record, error) {
	meta := s.schema.Get(key.Kind())
	if meta == nil {
		return nil, errors.New("kind not found")
	}

	query, args, err := builder.
		Select("*").
		From(meta.Table).
		Where(squirrel.Eq{"id": key.ID()}).
		ToSql()
	if err != nil {
		return nil, err
	}

	record := xdb.NewRecord(key)

	err = sqlitex.Execute(s.db, query, &sqlitex.ExecOptions{
		Args:       args,
		ResultFunc: s.scanRecord(record),
	})
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *SQLiteStore) scanRecord(record *xdb.Record) func(stmt *sqlite.Stmt) error {
	return func(stmt *sqlite.Stmt) error {
		meta := s.schema.Get(record.Key().Kind())

		for i := 0; i < stmt.ColumnCount(); i++ {
			name := stmt.ColumnName(i)

			if name == "id" {
				continue
			}

			attr := meta.Get(name)
			if attr == nil {
				return errors.New("attribute not found")
			}

			var t *xdb.Tuple

			switch attr.Type {
			case schema.String:
				t = xdb.NewTuple(record.Key(), name, stmt.ColumnText(i))
			case schema.Int:
				t = xdb.NewTuple(record.Key(), name, stmt.ColumnInt64(i))
			case schema.Float:
				t = xdb.NewTuple(record.Key(), name, stmt.ColumnFloat(i))
			case schema.Bool:
				t = xdb.NewTuple(record.Key(), name, stmt.ColumnInt(i) == 1)
			default:
				return errors.New("unknown type")
			}

			record.Set(t)
		}

		return nil
	}
}

func (s *SQLiteStore) DeleteRecord(ctx context.Context, key *xdb.Key) error {
	meta := s.schema.Get(key.Kind())
	if meta == nil {
		return errors.New("kind not found")
	}

	query, args, err := builder.
		Delete(meta.Table).
		Where(squirrel.Eq{"id": key.ID()}).
		ToSql()
	if err != nil {
		return err
	}

	err = sqlitex.Execute(s.db, query, &sqlitex.ExecOptions{Args: args})
	if err != nil {
		return err
	}

	return nil
}

func updateRecordQuery(tbl string, record *xdb.Record) (string, []interface{}, error) {
	cols := make([]string, 0)
	vals := make([]interface{}, 0)
	upserts := make([]string, 0)

	cols = append(cols, "id")
	vals = append(vals, record.Key().ID())

	tuples := record.Tuples()

	for _, tuple := range tuples {
		cols = append(cols, tuple.Name())
		vals = append(vals, tuple.Value())
		upserts = append(upserts, tuple.Name()+" = excluded."+tuple.Name())
	}

	query := builder.Insert(tbl).
		Columns(cols...).
		Values(vals...).
		Suffix("ON CONFLICT (id) DO UPDATE SET " + strings.Join(upserts, ", "))

	return query.ToSql()
}
