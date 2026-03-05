package xdbredis

import (
	"context"

	"github.com/gojekfarm/xtools/errors"
	"github.com/redis/go-redis/v9"

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

	pipe := s.db.Pipeline()

	type cmdEntry struct {
		uri *core.URI
		cmd *redis.StringCmd
	}

	var allEntries []cmdEntry
	var allMissed []*core.URI

	for _, schemaURIs := range grouped {
		schemaURI := schemaURIs[0].SchemaURI()

		if _, err := s.getSchemaFromCache(schemaURI); err != nil {
			allMissed = append(allMissed, schemaURIs...)
			continue
		}

		for _, uri := range schemaURIs {
			cmd := pipe.HGet(ctx, uri.ID().String(), uri.Attr().String())
			allEntries = append(allEntries, cmdEntry{uri: uri, cmd: cmd})
		}
	}

	if len(allEntries) == 0 {
		return nil, allMissed, nil
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	var allTuples []*core.Tuple

	for _, entry := range allEntries {
		err := entry.cmd.Err()
		if errors.Is(err, redis.Nil) {
			allMissed = append(allMissed, entry.uri)
			continue
		} else if err != nil {
			return nil, nil, err
		}

		val, err := s.codec.DecodeValue([]byte(entry.cmd.Val()))
		if err != nil {
			return nil, nil, err
		}

		uri := entry.uri
		path := uri.NS().String() + "/" + uri.Schema().String() + "/" + uri.ID().String()
		tuple := core.NewTuple(path, uri.Attr().String(), val.Unwrap())
		allTuples = append(allTuples, tuple)
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

	for _, schemaTuples := range grouped {
		if err := s.putTuplesForSchema(ctx, schemaTuples); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) putTuplesForSchema(ctx context.Context, tuples []*core.Tuple) error {
	schemaURI := tuples[0].SchemaURI()

	schemaDef, err := s.getSchemaFromCache(schemaURI)
	if err != nil {
		return err
	}

	switch schemaDef.Mode {
	case schema.ModeFlexible, schema.ModeStrict:
		if err := schema.ValidateTuples(schemaDef, tuples); err != nil {
			return err
		}
	case schema.ModeDynamic:
		newFields, err := schema.InferFields(schemaDef, tuples)
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

		if err := schema.ValidateTuples(schemaDef, tuples); err != nil {
			return err
		}
	}

	return s.writeTuples(ctx, tuples)
}

func (s *Store) writeTuples(ctx context.Context, tuples []*core.Tuple) error {
	pipe := s.db.Pipeline()

	for _, tuple := range tuples {
		hmval, err := s.codec.EncodeValue(tuple.Value())
		if err != nil {
			return err
		}
		pipe.HSet(ctx, tuple.ID().String(), tuple.Attr().String(), hmval)
	}

	_, err := pipe.Exec(ctx)

	return err
}

// DeleteTuples removes individual tuples from the store.
func (s *Store) DeleteTuples(ctx context.Context, uris []*core.URI) error {
	if len(uris) == 0 {
		return nil
	}

	pipe := s.db.Pipeline()

	for _, uri := range uris {
		pipe.HDel(ctx, uri.ID().String(), uri.Attr().String())
	}

	_, err := pipe.Exec(ctx)

	return err
}

func (s *Store) persistSchema(ctx context.Context, def *schema.Def) error {
	ns := def.NS.String()
	name := def.Name

	jsonSchema, err := schema.WriteToJSON(def)
	if err != nil {
		return err
	}

	pipe := s.db.Pipeline()
	pipe.Set(ctx, schemaKey(ns, name), jsonSchema, 0)
	pipe.SAdd(ctx, nsIndexKey, ns)
	pipe.SAdd(ctx, schemaIndexKey(ns), name)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	if s.schemas[ns] == nil {
		s.schemas[ns] = make(map[string]*schema.Def)
	}
	s.schemas[ns][name] = def
	s.mu.Unlock()

	return nil
}
