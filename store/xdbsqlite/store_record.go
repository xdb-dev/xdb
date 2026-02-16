package xdbsqlite

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/x"
)

// GetRecords retrieves records by URIs.
func (s *Store) GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error) {
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

	var allRecords []*core.Record
	var allMissed []*core.URI

	for _, schemaURIs := range grouped {
		schemaURI := schemaURIs[0].SchemaURI()

		schemaDef, err := s.getSchemaFromCache(schemaURI)
		if err != nil {
			allMissed = append(allMissed, schemaURIs...)
			continue
		}

		ids := x.Map(schemaURIs, func(uri *core.URI) string {
			return uri.ID().String()
		})

		var records []*core.Record
		var missed []*core.URI

		switch schemaDef.Mode {
		case schema.ModeFlexible:
			driver := NewKVDriverTx(tx, schemaDef)
			records, missed, err = driver.GetRecords(ctx, ids)
			if err != nil {
				return nil, nil, err
			}
		case schema.ModeStrict, schema.ModeDynamic:
			driver := NewSQLDriverTx(tx, schemaDef)
			records, missed, err = driver.GetRecords(ctx, ids)
			if err != nil {
				return nil, nil, err
			}
		}

		allRecords = append(allRecords, records...)
		allMissed = append(allMissed, missed...)
	}

	return allRecords, allMissed, nil
}

// PutRecords saves records to the store.
func (s *Store) PutRecords(ctx context.Context, records []*core.Record) error {
	if len(records) == 0 {
		return nil
	}

	grouped := x.GroupBy(records, func(r *core.Record) string {
		return r.SchemaURI().String()
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, schemaRecords := range grouped {
		schemaURI := schemaRecords[0].SchemaURI()

		schemaDef, err := s.getSchemaFromCache(schemaURI)
		if err != nil {
			return err
		}

		switch schemaDef.Mode {
		case schema.ModeFlexible:
			err = schema.ValidateRecords(schemaDef, schemaRecords)
			if err != nil {
				return err
			}
			driver := NewKVDriverTx(tx, schemaDef)
			if err := driver.PutRecords(ctx, schemaRecords); err != nil {
				return err
			}
		case schema.ModeStrict:
			err = schema.ValidateRecords(schemaDef, schemaRecords)
			if err != nil {
				return err
			}
			driver := NewSQLDriverTx(tx, schemaDef)
			if err := driver.PutRecords(ctx, schemaRecords); err != nil {
				return err
			}
		case schema.ModeDynamic:
			driver := NewSQLDriverTx(tx, schemaDef)
			schemaDef, err = driver.AddDynamicFields(ctx, schemaRecords)
			if err != nil {
				return err
			}

			s.mu.Lock()
			s.schemas[schemaURI.NS().String()][schemaURI.Schema().String()] = schemaDef
			s.mu.Unlock()

			err = schema.ValidateRecords(schemaDef, schemaRecords)
			if err != nil {
				return err
			}

			if err := driver.PutRecords(ctx, schemaRecords); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// DeleteRecords removes records from the store.
func (s *Store) DeleteRecords(ctx context.Context, uris []*core.URI) error {
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

		ids := x.Map(schemaURIs, func(uri *core.URI) string {
			return uri.ID().String()
		})

		switch schemaDef.Mode {
		case schema.ModeFlexible:
			driver := NewKVDriverTx(tx, schemaDef)
			if err := driver.DeleteRecords(ctx, ids); err != nil {
				return err
			}
		case schema.ModeStrict, schema.ModeDynamic:
			driver := NewSQLDriverTx(tx, schemaDef)
			if err := driver.DeleteRecords(ctx, ids); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
