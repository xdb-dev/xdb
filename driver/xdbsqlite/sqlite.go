package xdbsqlite

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/gojekfarm/xtools/errors"
	"github.com/spf13/cast"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/registry"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

var (
	_ driver.TupleReader  = (*SQLStore)(nil)
	_ driver.TupleWriter  = (*SQLStore)(nil)
	_ driver.RecordReader = (*SQLStore)(nil)
	_ driver.RecordWriter = (*SQLStore)(nil)
)

var (
	ErrSchemaNotFound   = errors.New("xdb/driver/xdbsqlite: schema not found")
	ErrUnknownAttr      = errors.New("xdb/driver/xdbsqlite: unknown attribute")
	ErrUnsupportedValue = errors.New("xdb/driver/xdbsqlite: unsupported value type")
)

// SQLStore is a store that uses SQLite as the underlying database.
type SQLStore struct {
	db       *sql.DB
	registry *registry.Registry
}

// New creates a new SQLite driver
func NewSQLStore(db *sql.DB, registry *registry.Registry) *SQLStore {
	return &SQLStore{db: db, registry: registry}
}

// GetTuples gets tuples from the SQLite database.
func (s *SQLStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	grouped := x.GroupAttrs(keys...)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	defer tx.Rollback()

	tupleMap := map[string]*types.Tuple{}

	for kind, rows := range grouped {
		schema := s.registry.Get(kind)
		if schema == nil {
			return nil, nil, errors.Wrap(ErrSchemaNotFound, "kind", kind)
		}

		for id, attrs := range rows {
			cols := x.Map(attrs, func(v string) any { return v })

			query, args, err := goqu.Select(cols...).
				From(kind).
				Where(goqu.L("id = ?", id)).
				ToSQL()

			rows, err := tx.QueryContext(ctx, query, args...)
			if err != nil {
				return nil, nil, err
			}

			defer rows.Close()

			for rows.Next() {
				dest := x.Map(attrs, func(a string) any {
					attrSchema := schema.GetAttribute(a)
					if attrSchema == nil {
						return nil
					}

					return &sqlValue{attr: attrSchema}
				})

				err := rows.Scan(dest...)
				if err != nil {
					return nil, nil, err
				}

				for i, attr := range attrs {
					val := dest[i].(*sqlValue).value
					if val == nil {
						continue
					}

					tuple := types.NewTuple(kind, id, attr, val)
					tupleMap[tuple.Key().String()] = tuple
				}
			}
		}
	}

	tuples := make([]*types.Tuple, 0)
	missing := make([]*types.Key, 0)

	for _, key := range keys {
		tuple, ok := tupleMap[key.String()]
		if !ok {
			missing = append(missing, key)
			continue
		}

		tuples = append(tuples, tuple)
	}

	return tuples, missing, nil
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
		schema := s.registry.Get(kind)
		if schema == nil {
			return errors.Wrap(ErrSchemaNotFound, "kind", kind)
		}

		for id, tuples := range rows {
			insertRecord := goqu.Record{"id": id}
			updateRecord := goqu.Record{}

			for _, tuple := range tuples {
				attr := tuple.Attr()
				val := &sqlValue{
					attr:  schema.GetAttribute(attr),
					value: tuple.Value(),
				}

				insertRecord[attr] = val
				updateRecord[attr] = goqu.I("EXCLUDED." + attr)
			}

			insertQuery := goqu.Insert(kind).
				Prepared(true).
				Rows(insertRecord).
				OnConflict(goqu.DoUpdate("id", updateRecord))

			query, args, err := insertQuery.ToSQL()
			if err != nil {
				return err
			}

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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	grouped := x.GroupAttrs(keys...)

	for kind, rows := range grouped {
		for id, attrs := range rows {
			updateRecord := goqu.Record{}

			for _, attr := range attrs {
				updateRecord[attr] = nil
			}

			query, args, err := goqu.Update(kind).
				Where(goqu.L("id = ?", id)).
				Set(updateRecord).
				ToSQL()

			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// GetRecords gets records from the SQLite database.
func (s *SQLStore) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
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

		schema := s.registry.Get(kind)
		if schema == nil {
			return nil, nil, errors.Wrap(ErrSchemaNotFound, "kind", kind)
		}

		attrs := x.Map(schema.Attributes, func(attr types.Attribute) any {
			return attr.Name
		})

		query, args, err := goqu.Select(attrs...).
			From(kind).
			Where(goqu.Ex{
				"id": ids,
			}).
			ToSQL()

		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, nil, err
		}

		defer rows.Close()

		for rows.Next() {
			values := x.Map(attrs, func(attr any) any {
				attrSchema := schema.GetAttribute(attr.(string))

				return &sqlValue{attr: attrSchema}
			})

			err := rows.Scan(values...)
			if err != nil {
				return nil, nil, err
			}

			pk := x.Filter(values, func(v any) bool {
				return v.(*sqlValue).attr.PrimaryKey
			})

			id := pk[0].(*sqlValue).value.String()

			record := types.NewRecord(kind, id)

			for i := range attrs {
				v := values[i].(*sqlValue)
				record.Set(v.attr.Name, v.value)
			}

			recordsMap[record.Key().String()] = record
		}
	}

	records := make([]*types.Record, 0)
	missing := make([]*types.Key, 0)

	for _, key := range keys {
		record, ok := recordsMap[key.String()]
		if !ok {
			missing = append(missing, key)
		}

		records = append(records, record)
	}

	return records, missing, nil
}

// PutRecords puts records into the SQLite database.
func (s *SQLStore) PutRecords(ctx context.Context, records []*types.Record) error {
	tuples := make([]*types.Tuple, 0, len(records))

	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}

	return s.PutTuples(ctx, tuples)
}

// DeleteRecords deletes records from the SQLite database.
func (s *SQLStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	if len(keys) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
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

		query, args, err := goqu.Delete(kind).
			Where(goqu.Ex{
				"id": ids,
			}).
			ToSQL()

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

type sqlValue struct {
	attr  *types.Attribute
	value types.Value
}

// Value returns a SQLite compatible type for a xdb/types.Value.
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
func (s *sqlValue) Value() (sqldriver.Value, error) {
	switch v := s.value.(type) {
	case types.Bool:
		if v {
			return 1, nil
		}

		return 0, nil
	case types.Int64:
		return int64(v), nil
	case types.Uint64:
		return uint64(v), nil
	case types.Float64:
		return float64(v), nil
	case types.String:
		return string(v), nil
	case types.Bytes:
		return v, nil
	case types.Time:
		return time.Time(v).UnixMilli(), nil
	case *types.Array:
		at := v.Type().(types.ArrayType)
		switch at.ValueType().ID() {
		case types.TypeInteger:
			// Convert integers to strings to maintain precision
			values := x.Map(v.Values(), func(v types.Value) string {
				return fmt.Sprintf("%d", v)
			})

			return json.Marshal(values)
		default:
			return json.Marshal(v.Values())
		}
	default:
		return nil, errors.Wrap(ErrUnsupportedValue, "type", s.value.Type().Name())
	}
}

// Scan converts a SQLite compatible type to a xdb/types.Value.
func (s *sqlValue) Scan(src any) error {
	if src == nil {
		return nil
	}
	switch s.attr.Type.ID() {
	case types.TypeArray:
		var decoded []any
		if err := json.Unmarshal(src.([]byte), &decoded); err != nil {
			return err
		}

		s.value = castValue(decoded, s.attr.Type)
	default:
		s.value = castValue(src, s.attr.Type)
	}

	return nil
}

func castValue(src any, typ types.Type) types.Value {
	switch typ.ID() {
	case types.TypeBoolean:
		return types.Bool(cast.ToBool(src))
	case types.TypeInteger:
		return types.Int64(cast.ToInt64(src))
	case types.TypeUnsigned:
		return types.Uint64(cast.ToUint64(src))
	case types.TypeFloat:
		return types.Float64(cast.ToFloat64(src))
	case types.TypeString:
		return types.String(cast.ToString(src))
	case types.TypeBytes:
		return types.Bytes(src.([]byte))
	case types.TypeTime:
		return types.Time(time.UnixMilli(cast.ToInt64(src)))
	case types.TypeArray:
		at := typ.(types.ArrayType)
		values := x.Map(src.([]any), func(v any) types.Value {
			return castValue(v, at.ValueType())
		})

		return types.NewArray(at.ValueType(), values...)
	}

	return nil
}
