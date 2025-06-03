package xdbsqlite

import (
	"context"
	"database/sql"
	"encoding/base64"
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
	ErrSchemaNotFound = errors.New("xdb/driver/xdbsqlite: schema not found")
	ErrUnknownAttr    = errors.New("xdb/driver/xdbsqlite: unknown attribute")
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

					return &sqlValue{Attribute: *attrSchema}
				})

				err := rows.Scan(dest...)
				if err != nil {
					return nil, nil, err
				}

				for i, attr := range attrs {
					val := dest[i].(*sqlValue).Value
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
		for id, tuples := range rows {
			insertRecord := goqu.Record{"id": id}
			updateRecord := goqu.Record{}

			for _, tuple := range tuples {
				attr := tuple.Attr()
				val, err := s.encodeValue(tuple.Value())
				if err != nil {
					return err
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
func (s *SQLStore) encodeValue(value *types.Value) (any, error) {
	if value.Repeated() {
		switch value.TypeID() {
		case types.TypeInteger:
			// convert to string array to maintain precision
			// integer can be lossy when converted to float in JSON
			return json.Marshal(value.ToStringSlice())
		default:
			return json.Marshal(value.Unwrap())
		}
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
		return value.ToTime().UnixMilli(), nil
	case types.TypePoint:
		return value.ToPoint(), nil
	}

	return nil, fmt.Errorf("unsupported value type: %s", value.TypeID())
}

type sqlValue struct {
	types.Attribute
	Value *types.Value
}

func (v *sqlValue) Scan(src any) error {
	if src == nil {
		return nil
	}

	var decoded any

	if v.Attribute.Repeated {
		var val []any
		if err := json.Unmarshal(src.([]byte), &val); err != nil {
			return err
		}

		switch v.Attribute.Type {
		case types.TypeString:
			decoded = x.CastArray(val, cast.ToString)
		case types.TypeInteger:
			decoded = x.CastArray(val, cast.ToInt64)
		case types.TypeFloat:
			decoded = x.CastArray(val, cast.ToFloat64)
		case types.TypeBoolean:
			decoded = x.CastArray(val, cast.ToBool)
		case types.TypeBytes:
			decoded = x.Map(val, func(v any) []byte {
				decoded, err := base64.StdEncoding.DecodeString(v.(string))
				if err != nil {
					return nil
				}

				return decoded
			})
		case types.TypeTime:
			decoded = x.CastArray(val, func(v any) time.Time {
				return time.UnixMilli(cast.ToInt64(v))
			})
		}
	} else {
		switch v.Attribute.Type {
		case types.TypeString:
			decoded = cast.ToString(src)
		case types.TypeInteger:
			decoded = cast.ToInt64(src)
		case types.TypeFloat:
			decoded = cast.ToFloat64(src)
		case types.TypeBoolean:
			decoded = cast.ToBool(src)
		case types.TypeBytes:
			decoded = x.ToBytes(src)
		case types.TypeTime:
			decoded = time.UnixMilli(cast.ToInt64(src))
		}
	}

	if decoded != nil {
		v.Value = types.NewValue(decoded)
	}

	return nil
}
