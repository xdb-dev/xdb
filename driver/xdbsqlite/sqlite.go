package xdbsqlite

// import (
// 	"context"
// 	"database/sql"
// 	"fmt"

// 	"github.com/doug-martin/goqu/v9"

// 	// Register SQLite3 dialect for goqu
// 	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
// 	"github.com/gojekfarm/xtools/errors"

// 	"github.com/xdb-dev/xdb/core"
// 	"github.com/xdb-dev/xdb/driver"
// 	"github.com/xdb-dev/xdb/registry"
// 	"github.com/xdb-dev/xdb/x"
// )

// var (
// 	_ driver.TupleReader  = (*SQLStore)(nil)
// 	_ driver.TupleWriter  = (*SQLStore)(nil)
// 	_ driver.RecordReader = (*SQLStore)(nil)
// 	_ driver.RecordWriter = (*SQLStore)(nil)
// )

// var (
// 	// ErrSchemaNotFound is returned when a schema is not found in the registry.
// 	ErrSchemaNotFound = errors.New("xdb/driver/xdbsqlite: schema not found")
// 	// ErrUnknownAttr is returned when an unknown attribute is encountered in the schema.
// 	ErrUnknownAttr = errors.New("xdb/driver/xdbsqlite: unknown attribute")
// 	// ErrUnsupportedValue is returned when an unsupported value type is encountered.
// 	ErrUnsupportedValue = errors.New("xdb/driver/xdbsqlite: unsupported value type")
// )

// // SQLStore is a store that uses SQLite as the underlying database.
// type SQLStore struct {
// 	db       *sql.DB
// 	registry *registry.Registry
// }

// // NewSQLStore creates a new SQLStore using the provided database and registry.
// func NewSQLStore(db *sql.DB, registry *registry.Registry) *SQLStore {
// 	return &SQLStore{db: db, registry: registry}
// }

// // GetTuples gets tuples from the SQLite database.
// func (s *SQLStore) GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error) {
// 	grouped := x.GroupAttrs(keys...)

// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	defer tx.Rollback()

// 	tupleMap := map[string]*core.Tuple{}

// 	for kind, rows := range grouped {
// 		schema := s.registry.Get(kind)
// 		if schema == nil {
// 			return nil, nil, errors.Wrap(ErrSchemaNotFound, "kind", kind)
// 		}

// 		for id, attrs := range rows {
// 			cols := x.Map(attrs, func(v string) any { return v })

// 			query, args, err := goqu.Select(cols...).
// 				From(kind).
// 				Where(goqu.L("id = ?", id)).
// 				ToSQL()

// 			rows, err := tx.QueryContext(ctx, query, args...)
// 			if err != nil {
// 				return nil, nil, err
// 			}

// 			defer rows.Close()

// 			for rows.Next() {
// 				dest := x.Map(attrs, func(a string) any {
// 					attrSchema := schema.GetAttribute(a)
// 					if attrSchema == nil {
// 						return nil
// 					}

// 					return &sqlValue{attr: attrSchema}
// 				})

// 				err := rows.Scan(dest...)
// 				if err != nil {
// 					return nil, nil, err
// 				}

// 				for i, attr := range attrs {
// 					val := dest[i].(*sqlValue).value
// 					if val == nil {
// 						continue
// 					}

// 					tuple := core.NewTuple(kind, id, attr, val)
// 					tupleMap[tuple.Key().String()] = tuple
// 				}
// 			}
// 		}
// 	}

// 	tuples := make([]*core.Tuple, 0)
// 	missing := make([]*core.Key, 0)

// 	for _, key := range keys {
// 		tuple, ok := tupleMap[key.String()]
// 		if !ok {
// 			missing = append(missing, key)
// 			continue
// 		}

// 		tuples = append(tuples, tuple)
// 	}

// 	return tuples, missing, nil
// }

// // PutTuples puts tuples into the SQLite database.
// func (s *SQLStore) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
// 	grouped := x.GroupTuples(tuples...)

// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	defer tx.Rollback()

// 	for kind, rows := range grouped {
// 		schema := s.registry.Get(kind)
// 		if schema == nil {
// 			return errors.Wrap(ErrSchemaNotFound, "kind", kind)
// 		}

// 		for id, tuples := range rows {
// 			insertRecord := goqu.Record{"id": id}
// 			updateRecord := goqu.Record{}

// 			for _, tuple := range tuples {
// 				attr := tuple.Attr()
// 				sqlVal := &sqlValue{
// 					attr:  schema.GetAttribute(attr),
// 					value: tuple.Value(),
// 				}

// 				encoded, err := sqlVal.Value()
// 				if err != nil {
// 					return err
// 				}

// 				insertRecord[attr] = encoded
// 				updateRecord[attr] = goqu.I("EXCLUDED." + attr)
// 			}

// 			insertQuery := goqu.Insert(kind).
// 				Prepared(true).
// 				Rows(insertRecord).
// 				OnConflict(goqu.DoUpdate("id", updateRecord))

// 			query, args, err := insertQuery.ToSQL()
// 			if err != nil {
// 				return err
// 			}

// 			fmt.Println(query, args)
// 			_, err = tx.ExecContext(ctx, query, args...)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return tx.Commit()
// }

// // DeleteTuples deletes tuples from the SQLite database.
// func (s *SQLStore) DeleteTuples(ctx context.Context, keys []*core.Key) error {
// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	defer tx.Rollback()

// 	grouped := x.GroupAttrs(keys...)

// 	for kind, rows := range grouped {
// 		for id, attrs := range rows {
// 			updateRecord := goqu.Record{}

// 			for _, attr := range attrs {
// 				updateRecord[attr] = nil
// 			}

// 			query, args, err := goqu.Update(kind).
// 				Where(goqu.L("id = ?", id)).
// 				Set(updateRecord).
// 				ToSQL()

// 			_, err = tx.ExecContext(ctx, query, args...)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return tx.Commit()
// }

// // GetRecords gets records from the SQLite database.
// func (s *SQLStore) GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error) {
// 	if len(keys) == 0 {
// 		return nil, nil, nil
// 	}

// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	defer tx.Rollback()

// 	grouped := x.GroupBy(keys, func(key *core.Key) string {
// 		return key.Kind()
// 	})

// 	recordsMap := make(map[string]*core.Record)

// 	for kind, keys := range grouped {
// 		ids := x.Map(keys, func(key *core.Key) string {
// 			return key.ID()
// 		})

// 		schema := s.registry.Get(kind)
// 		if schema == nil {
// 			return nil, nil, errors.Wrap(ErrSchemaNotFound, "kind", kind)
// 		}

// 		attrs := x.Map(schema.Attributes, func(attr core.Attribute) any {
// 			return attr.Name
// 		})

// 		query, args, err := goqu.Select(attrs...).
// 			From(kind).
// 			Where(goqu.Ex{
// 				"id": ids,
// 			}).
// 			ToSQL()

// 		rows, err := tx.QueryContext(ctx, query, args...)
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		defer rows.Close()

// 		for rows.Next() {
// 			values := x.Map(attrs, func(attr any) any {
// 				attrSchema := schema.GetAttribute(attr.(string))

// 				return &sqlValue{attr: attrSchema}
// 			})

// 			err := rows.Scan(values...)
// 			if err != nil {
// 				return nil, nil, err
// 			}

// 			pk := x.Filter(values, func(v any) bool {
// 				return v.(*sqlValue).attr.PrimaryKey
// 			})

// 			id := pk[0].(*sqlValue).value.String()

// 			record := core.NewRecord(kind, id)

// 			for i := range attrs {
// 				v := values[i].(*sqlValue)
// 				record.Set(v.attr.Name, v.value)
// 			}

// 			recordsMap[record.Key().String()] = record
// 		}
// 	}

// 	records := make([]*core.Record, 0)
// 	missing := make([]*core.Key, 0)

// 	for _, key := range keys {
// 		record, ok := recordsMap[key.String()]
// 		if !ok {
// 			missing = append(missing, key)
// 		}

// 		records = append(records, record)
// 	}

// 	return records, missing, nil
// }

// // PutRecords puts records into the SQLite database.
// func (s *SQLStore) PutRecords(ctx context.Context, records []*core.Record) error {
// 	tuples := make([]*core.Tuple, 0, len(records))

// 	for _, record := range records {
// 		tuples = append(tuples, record.Tuples()...)
// 	}

// 	return s.PutTuples(ctx, tuples)
// }

// // DeleteRecords deletes records from the SQLite database.
// func (s *SQLStore) DeleteRecords(ctx context.Context, keys []*core.Key) error {
// 	if len(keys) == 0 {
// 		return nil
// 	}

// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	defer tx.Rollback()

// 	grouped := x.GroupBy(keys, func(key *core.Key) string {
// 		return key.Kind()
// 	})

// 	for kind, keys := range grouped {
// 		ids := x.Map(keys, func(key *core.Key) string {
// 			return key.ID()
// 		})

// 		query, args, err := goqu.Delete(kind).
// 			Where(goqu.Ex{
// 				"id": ids,
// 			}).
// 			ToSQL()

// 		_, err = tx.ExecContext(ctx, query, args...)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return tx.Commit()
// }
