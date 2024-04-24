package sqlite

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

	query, args, err := updateRecordQuery(meta, record)
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

	records := make([]*xdb.Record, 0)

	err = sqlitex.Execute(s.db, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: scanRecords(meta, func(r *xdb.Record) {
			records = append(records, r)
		}),
	})
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, errors.New("record not found")
	} else if len(records) > 1 {
		return nil, errors.New("multiple records found")
	}

	return records[0], nil
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

func (s *SQLiteStore) QueryRecords(ctx context.Context, q *xdb.Query) (*xdb.RecordSet, error) {
	meta := s.schema.Get(q.Kind())
	if meta == nil {
		return nil, errors.New("kind not found")
	}

	query, args, err := buildSelectQuery(meta, q)
	if err != nil {
		return nil, err
	}

	records := make([]*xdb.Record, 0)

	err = sqlitex.Execute(s.db, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: scanRecords(meta, func(record *xdb.Record) {
			records = append(records, record)
		}),
	})
	if err != nil {
		return nil, err
	}

	rs := xdb.NewResultSet(records, 0, 0)

	return rs, nil
}

func buildSelectQuery(meta *schema.Record, q *xdb.Query) (string, []interface{}, error) {
	attrs := q.GetAttributes()
	conditions := q.GetConditions()
	orderBy := q.GetOrderBy()
	skip := q.GetSkip()
	limit := q.GetLimit()

	sqlQuery := builder.Select("*").From(meta.Table)

	if len(attrs) > 0 {
		sqlQuery = sqlQuery.Columns(attrs...)
	}

	clauses := make(squirrel.And, 0)

	for attr, conds := range conditions {
		for _, cond := range conds {
			clauses = append(clauses, toSQLClause(attr, cond))
		}
	}

	sqlQuery = sqlQuery.Where(clauses)

	if len(orderBy) > 0 {
		for i := 0; i < len(orderBy); i += 2 {
			sqlQuery = sqlQuery.OrderBy(orderBy[i] + " " + orderBy[i+1])
		}
	}

	if skip > 0 {
		sqlQuery = sqlQuery.Offset(uint64(skip))
	}

	if limit > 0 {
		sqlQuery = sqlQuery.Limit(uint64(limit))
	}

	return sqlQuery.ToSql()
}

func toSQLClause(attr string, cond *xdb.Condition) squirrel.Sqlizer {
	switch cond.Op() {
	case xdb.OpEq:
		return squirrel.Eq{attr: cond.Value()}
	case xdb.OpNe:
		return squirrel.NotEq{attr: cond.Value()}
	case xdb.OpGt:
		return squirrel.Gt{attr: cond.Value()}
	case xdb.OpGe:
		return squirrel.GtOrEq{attr: cond.Value()}
	case xdb.OpLt:
		return squirrel.Lt{attr: cond.Value()}
	case xdb.OpLe:
		return squirrel.LtOrEq{attr: cond.Value()}
	case xdb.OpIn:
		return squirrel.Eq{attr: cond.ValueList()}
	case xdb.OpNotIn:
		return squirrel.NotEq{attr: cond.ValueList()}
	case xdb.OpSearch:
		return squirrel.Like{attr: fmt.Sprintf("%%%s%%", cond.Value())}
	}

	return nil
}

func updateRecordQuery(meta *schema.Record, record *xdb.Record) (string, []interface{}, error) {
	cols := make([]string, 0)
	vals := make([]interface{}, 0)
	upserts := make([]string, 0)

	cols = append(cols, "id")
	vals = append(vals, record.Key().ID())

	tuples := record.Tuples()

	for _, tuple := range tuples {
		attr := meta.Get(tuple.Name())
		cols = append(cols, tuple.Name())

		switch attr.Type {
		case schema.Time:
			vals = append(vals, tuple.Value().(time.Time).Unix())
		default:
			vals = append(vals, tuple.Value())
		}

		upserts = append(upserts, tuple.Name()+" = excluded."+tuple.Name())
	}

	query := builder.Insert(meta.Table).
		Columns(cols...).
		Values(vals...).
		Suffix("ON CONFLICT (id) DO UPDATE SET " + strings.Join(upserts, ", "))

	return query.ToSql()
}

func scanRecords(meta *schema.Record, fn func(*xdb.Record)) func(stmt *sqlite.Stmt) error {
	return func(stmt *sqlite.Stmt) error {
		id := stmt.ColumnText(stmt.ColumnIndex("id"))
		key := xdb.NewKey(meta.Kind, id)
		record := xdb.NewRecord(key)

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
			case schema.Time:
				ts := time.Unix(stmt.ColumnInt64(i), 0)

				t = xdb.NewTuple(record.Key(), name, ts)
			default:
				return errors.New("unknown type")
			}

			record.Set(t)
		}

		fn(record)

		return nil
	}
}
