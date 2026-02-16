package xdbsqlite

import (
	"context"
	"database/sql"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/x"
)

// GetTuples retrieves individual tuples by URIs.
func (s *Store) GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error) {
	if len(uris) == 0 {
		return nil, nil, nil
	}

	grouped := x.GroupBy(uris, func(uri *core.URI) string {
		return uri.NS().String() + "/" + uri.Schema().String()
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var allTuples []*core.Tuple
	var allMissed []*core.URI

	for _, schemaURIs := range grouped {
		schemaURI := schemaURIs[0].SchemaURI()

		schemaDef, err := s.getSchemaFromCache(schemaURI)
		if err != nil {
			allMissed = append(allMissed, schemaURIs...)
			continue
		}

		switch schemaDef.Mode {
		case schema.ModeFlexible:
			keys := x.Map(schemaURIs, func(uri *core.URI) string {
				return uri.ID().String() + "#" + uri.Attr().String()
			})
			driver := NewKVDriverTx(tx, schemaDef)
			tuples, missed, err := driver.GetTuples(ctx, keys)
			if err != nil {
				return nil, nil, err
			}
			allTuples = append(allTuples, tuples...)
			allMissed = append(allMissed, missed...)
		case schema.ModeStrict, schema.ModeDynamic:
			refs := x.Map(schemaURIs, func(uri *core.URI) TupleRef {
				return TupleRef{ID: uri.ID().String(), Attr: uri.Attr().String()}
			})
			driver := NewSQLDriverTx(tx, schemaDef)
			tuples, missed, err := driver.GetTuples(ctx, refs)
			if err != nil {
				return nil, nil, err
			}
			allTuples = append(allTuples, tuples...)
			allMissed = append(allMissed, missed...)
		}
	}

	return allTuples, allMissed, nil
}

// PutTuples saves individual tuples to the store.
func (s *Store) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	grouped := x.GroupBy(tuples, func(t *core.Tuple) string {
		return t.SchemaURI().String()
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, schemaTuples := range grouped {
		if err := s.putTuplesForSchema(ctx, tx, schemaTuples); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) putTuplesForSchema(ctx context.Context, tx *sql.Tx, tuples []*core.Tuple) error {
	schemaURI := tuples[0].SchemaURI()

	schemaDef, err := s.getSchemaFromCache(schemaURI)
	if err != nil {
		return err
	}

	switch schemaDef.Mode {
	case schema.ModeFlexible:
		if err := schema.ValidateTuples(schemaDef, tuples); err != nil {
			return err
		}
		return NewKVDriverTx(tx, schemaDef).PutTuples(ctx, tuples)
	case schema.ModeStrict:
		if err := schema.ValidateTuples(schemaDef, tuples); err != nil {
			return err
		}
		return NewSQLDriverTx(tx, schemaDef).PutTuples(ctx, tuples)
	case schema.ModeDynamic:
		driver := NewSQLDriverTx(tx, schemaDef)
		schemaDef, err = driver.AddDynamicFieldsFromTuples(ctx, tuples)
		if err != nil {
			return err
		}

		s.mu.Lock()
		s.schemas[schemaURI.NS().String()][schemaURI.Schema().String()] = schemaDef
		s.mu.Unlock()

		if err := schema.ValidateTuples(schemaDef, tuples); err != nil {
			return err
		}
		return driver.PutTuples(ctx, tuples)
	}

	return nil
}

// DeleteTuples removes individual tuples from the store.
func (s *Store) DeleteTuples(ctx context.Context, uris []*core.URI) error {
	if len(uris) == 0 {
		return nil
	}

	grouped := x.GroupBy(uris, func(uri *core.URI) string {
		return uri.NS().String() + "/" + uri.Schema().String()
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, schemaURIs := range grouped {
		schemaURI := schemaURIs[0].SchemaURI()

		schemaDef, err := s.getSchemaFromCache(schemaURI)
		if err != nil {
			continue
		}

		switch schemaDef.Mode {
		case schema.ModeFlexible:
			keys := x.Map(schemaURIs, func(uri *core.URI) string {
				return uri.ID().String() + "#" + uri.Attr().String()
			})
			driver := NewKVDriverTx(tx, schemaDef)
			if err := driver.DeleteTuples(ctx, keys); err != nil {
				return err
			}
		case schema.ModeStrict, schema.ModeDynamic:
			refs := x.Map(schemaURIs, func(uri *core.URI) TupleRef {
				return TupleRef{ID: uri.ID().String(), Attr: uri.Attr().String()}
			})
			driver := NewSQLDriverTx(tx, schemaDef)
			if err := driver.DeleteTuples(ctx, refs); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
