package xdbredis

import (
	"context"

	"github.com/gojekfarm/xtools/errors"
	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/x"
)

type recordCmdEntry struct {
	uri *core.URI
	cmd *redis.MapStringStringCmd
}

// GetRecords retrieves records by URIs.
func (s *Store) GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error) {
	if len(uris) == 0 {
		return nil, nil, nil
	}

	grouped := x.GroupBy(uris, func(uri *core.URI) string {
		return uri.NS().String() + "/" + uri.Schema().String()
	})

	pipe := s.db.Pipeline()

	var allEntries []recordCmdEntry
	var allMissed []*core.URI

	for _, schemaURIs := range grouped {
		schemaURI := schemaURIs[0].SchemaURI()

		if _, err := s.getSchemaFromCache(schemaURI); err != nil {
			allMissed = append(allMissed, schemaURIs...)
			continue
		}

		for _, uri := range schemaURIs {
			cmd := pipe.HGetAll(ctx, uri.ID().String())
			allEntries = append(allEntries, recordCmdEntry{uri: uri, cmd: cmd})
		}
	}

	if len(allEntries) == 0 {
		return nil, allMissed, nil
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	return s.collectRecords(allEntries, allMissed)
}

func (s *Store) collectRecords(entries []recordCmdEntry, missed []*core.URI) ([]*core.Record, []*core.URI, error) {
	records := make([]*core.Record, 0, len(entries))

	for _, entry := range entries {
		if err := entry.cmd.Err(); err != nil {
			return nil, nil, err
		}

		attrs := entry.cmd.Val()
		if len(attrs) == 0 {
			missed = append(missed, entry.uri)
			continue
		}

		uri := entry.uri
		record := core.NewRecord(uri.NS().String(), uri.Schema().String(), uri.ID().String())

		for attr, val := range attrs {
			decodedVal, err := s.codec.DecodeValue([]byte(val))
			if err != nil {
				return nil, nil, err
			}
			record.Set(attr, decodedVal)
		}

		records = append(records, record)
	}

	return records, missed, nil
}

// PutRecords saves records to the store.
func (s *Store) PutRecords(ctx context.Context, records []*core.Record) error {
	if len(records) == 0 {
		return nil
	}

	grouped := x.GroupBy(records, func(r *core.Record) string {
		return r.SchemaURI().String()
	})

	for _, schemaRecords := range grouped {
		schemaURI := schemaRecords[0].SchemaURI()

		schemaDef, err := s.getSchemaFromCache(schemaURI)
		if err != nil {
			return err
		}

		switch schemaDef.Mode {
		case schema.ModeFlexible, schema.ModeStrict:
			if err := schema.ValidateRecords(schemaDef, schemaRecords); err != nil {
				return err
			}
		case schema.ModeDynamic:
			var allTuples []*core.Tuple
			for _, record := range schemaRecords {
				allTuples = append(allTuples, record.Tuples()...)
			}

			newFields, err := schema.InferFields(schemaDef, allTuples)
			if err != nil {
				return err
			}

			if len(newFields) > 0 {
				updated := schemaDef.Clone()
				updated.AddFields(newFields...)

				if err := s.persistSchema(ctx, updated); err != nil {
					return err
				}

				schemaDef = updated
			}

			if err := schema.ValidateRecords(schemaDef, schemaRecords); err != nil {
				return err
			}
		}

		var tuples []*core.Tuple
		for _, record := range schemaRecords {
			tuples = append(tuples, record.Tuples()...)
		}

		if err := s.writeTuples(ctx, tuples); err != nil {
			return err
		}
	}

	return nil
}

// DeleteRecords removes records from the store.
func (s *Store) DeleteRecords(ctx context.Context, uris []*core.URI) error {
	if len(uris) == 0 {
		return nil
	}

	pipe := s.db.Pipeline()

	for _, uri := range uris {
		pipe.Del(ctx, uri.ID().String())
	}

	_, err := pipe.Exec(ctx)

	return err
}
