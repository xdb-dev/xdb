package xdbredis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// GetSchema retrieves a schema definition by URI.
func (s *Store) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	key := s.schemaKey(uri)

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("xdbredis: get schema: %w", err)
	}

	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("xdbredis: unmarshal schema: %w", err)
	}

	return &def, nil
}

// ListSchemas lists schemas, optionally scoped by namespace URI.
func (s *Store) ListSchemas(
	ctx context.Context,
	q *store.Query,
) (*store.Page[*schema.Def], error) {
	uri := q.URI

	if uri != nil {
		return s.listSchemasByNS(ctx, uri, q)
	}

	// All namespaces: get namespace list, then schemas for each.
	namespaces, err := s.client.SMembers(ctx, s.nsIndexKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: list namespaces: %w", err)
	}

	var defs []*schema.Def
	for _, ns := range namespaces {
		nsURI := core.New().NS(ns).MustURI()
		nsDefs, err := s.fetchSchemasByNS(ctx, nsURI)
		if err != nil {
			return nil, err
		}
		defs = append(defs, nsDefs...)
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].URI.Path() < defs[j].URI.Path()
	})

	return store.Paginate(defs, q), nil
}

// listSchemasByNS lists schemas for a single namespace.
func (s *Store) listSchemasByNS(
	ctx context.Context,
	uri *core.URI,
	q *store.Query,
) (*store.Page[*schema.Def], error) {
	defs, err := s.fetchSchemasByNS(ctx, uri)
	if err != nil {
		return nil, err
	}

	sort.Slice(defs, func(i, j int) bool {
		return defs[i].URI.Path() < defs[j].URI.Path()
	})

	return store.Paginate(defs, q), nil
}

// fetchSchemasByNS fetches all schema definitions for a namespace URI.
func (s *Store) fetchSchemasByNS(ctx context.Context, uri *core.URI) ([]*schema.Def, error) {
	idxKey := s.schemaIndexKey(uri)
	ns := uri.NS().String()

	names, err := s.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: list schema index: %w", err)
	}

	defs := make([]*schema.Def, 0, len(names))
	for _, name := range names {
		schemaURI := core.New().NS(ns).Schema(name).MustURI()
		key := s.schemaKey(schemaURI)

		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, fmt.Errorf("xdbredis: get schema %s: %w", name, err)
		}

		var def schema.Def
		if err := json.Unmarshal(data, &def); err != nil {
			return nil, fmt.Errorf("xdbredis: unmarshal schema %s: %w", name, err)
		}

		defs = append(defs, &def)
	}

	return defs, nil
}

// CreateSchema creates a new schema definition.
func (s *Store) CreateSchema(
	ctx context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	key := s.schemaKey(uri)

	// Check existence first.
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check schema exists: %w", err)
	}
	if exists > 0 {
		return store.ErrAlreadyExists
	}

	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal schema: %w", err)
	}

	// Atomic: set schema + update indexes.
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, key, data, 0)
	pipe.SAdd(ctx, s.schemaIndexKey(uri), uri.Schema().String())
	pipe.SAdd(ctx, s.nsIndexKey(), uri.NS().String())

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("xdbredis: create schema: %w", err)
	}

	return nil
}

// UpdateSchema updates an existing schema definition.
func (s *Store) UpdateSchema(
	ctx context.Context,
	uri *core.URI,
	def *schema.Def,
) error {
	key := s.schemaKey(uri)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check schema exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}

	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("xdbredis: marshal schema: %w", err)
	}

	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("xdbredis: update schema: %w", err)
	}

	return nil
}

// DeleteSchema deletes a schema and all its records.
func (s *Store) DeleteSchema(ctx context.Context, uri *core.URI) error {
	key := s.schemaKey(uri)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: check schema exists: %w", err)
	}
	if exists == 0 {
		return store.ErrNotFound
	}

	// Collect record keys to delete.
	idxKey := s.recordIndexKey(uri)
	recordIDs, err := s.client.SMembers(ctx, idxKey).Result()
	if err != nil {
		return fmt.Errorf("xdbredis: list record index: %w", err)
	}

	pipe := s.client.TxPipeline()

	// Delete all record keys.
	for _, id := range recordIDs {
		recURI := core.New().NS(uri.NS().String()).Schema(uri.Schema().String()).ID(id).MustURI()
		pipe.Del(ctx, s.recordKey(recURI))
	}

	// Delete schema, record index, and update schema/ns indexes.
	pipe.Del(ctx, key)
	pipe.Del(ctx, idxKey)
	pipe.SRem(ctx, s.schemaIndexKey(uri), uri.Schema().String())

	if _, execErr := pipe.Exec(ctx); execErr != nil {
		return fmt.Errorf("xdbredis: delete schema: %w", execErr)
	}

	// Clean up namespace index if no schemas remain.
	remaining, cardErr := s.client.SCard(ctx, s.schemaIndexKey(uri)).Result()
	if cardErr == nil && remaining == 0 {
		s.client.SRem(ctx, s.nsIndexKey(), uri.NS().String())
		s.client.Del(ctx, s.schemaIndexKey(uri))
	}

	return nil
}

// DeleteSchemaRecords is a no-op for Redis because [DeleteSchema]
// already removes all records belonging to the schema.
func (s *Store) DeleteSchemaRecords(_ context.Context, _ *core.URI) error {
	return nil
}
