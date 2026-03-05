package xdbredis

import (
	"context"
	"slices"

	"github.com/gojekfarm/xtools/errors"
	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

const (
	nsIndexKey = "xdb:ns"
)

func schemaKey(ns, name string) string {
	return "xdb:schema:" + ns + ":" + name
}

func schemaIndexKey(ns string) string {
	return "xdb:schemas:" + ns
}

func (s *Store) loadSchemas(ctx context.Context) error {
	s.mu.Lock()
	s.schemas = make(map[string]map[string]*schema.Def)
	s.mu.Unlock()

	namespaces, err := s.db.SMembers(ctx, nsIndexKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	for _, ns := range namespaces {
		names, err := s.db.SMembers(ctx, schemaIndexKey(ns)).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return err
		}

		s.mu.Lock()
		s.schemas[ns] = make(map[string]*schema.Def)
		s.mu.Unlock()

		for _, name := range names {
			data, err := s.db.Get(ctx, schemaKey(ns, name)).Bytes()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue
				}
				return err
			}

			def, err := schema.LoadFromJSON(data)
			if err != nil {
				return err
			}

			s.mu.Lock()
			s.schemas[ns][name] = def
			s.mu.Unlock()
		}
	}

	return nil
}

// GetSchema returns the schema definition for the given URI.
func (s *Store) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	if def, err := s.getSchemaFromCache(uri); err == nil {
		return def, nil
	}

	ns := uri.NS().String()
	name := uri.Schema().String()

	data, err := s.db.Get(ctx, schemaKey(ns, name)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	def, err := schema.LoadFromJSON(data)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if _, ok := s.schemas[ns]; !ok {
		s.schemas[ns] = make(map[string]*schema.Def)
	}
	s.schemas[ns][name] = def
	s.mu.Unlock()

	return def, nil
}

func (s *Store) getSchemaFromCache(uri *core.URI) (*schema.Def, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns := uri.NS().String()
	name := uri.Schema().String()

	if nsSchemas, ok := s.schemas[ns]; ok {
		if def, ok := nsSchemas[name]; ok {
			return def, nil
		}
	}

	return nil, store.ErrNotFound
}

// ListNamespaces returns all unique namespaces that contain schemas.
func (s *Store) ListNamespaces(_ context.Context) ([]*core.NS, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	namespaces := make([]*core.NS, 0, len(s.schemas))
	for nsStr := range s.schemas {
		ns, err := core.ParseNS(nsStr)
		if err != nil {
			continue
		}
		namespaces = append(namespaces, ns)
	}

	slices.SortFunc(namespaces, func(a, b *core.NS) int {
		if a.String() < b.String() {
			return -1
		}
		if a.String() > b.String() {
			return 1
		}
		return 0
	})

	return namespaces, nil
}

// ListSchemas returns all schema definitions in the given namespace.
func (s *Store) ListSchemas(_ context.Context, uri *core.URI) ([]*schema.Def, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns := uri.NS().String()
	nsSchemas, ok := s.schemas[ns]
	if !ok {
		return nil, nil
	}

	schemas := make([]*schema.Def, 0, len(nsSchemas))
	for _, def := range nsSchemas {
		schemas = append(schemas, def)
	}

	slices.SortFunc(schemas, func(a, b *schema.Def) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return schemas, nil
}

// PutSchema creates or updates a schema definition.
func (s *Store) PutSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	if err := validateSchemaURI(uri, def); err != nil {
		return err
	}

	ns := uri.NS().String()
	name := uri.Schema().String()

	existing, _ := s.getSchemaFromCache(uri)
	if existing != nil {
		if existing.Mode != def.Mode {
			return store.ErrSchemaModeChanged
		}
		if err := validateFieldChanges(existing, def); err != nil {
			return err
		}
	}

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

// DeleteSchema removes a schema definition.
func (s *Store) DeleteSchema(ctx context.Context, uri *core.URI) error {
	ns := uri.NS().String()
	name := uri.Schema().String()

	pipe := s.db.Pipeline()
	pipe.Del(ctx, schemaKey(ns, name))
	pipe.SRem(ctx, schemaIndexKey(ns), name)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	remaining, err := s.db.SCard(ctx, schemaIndexKey(ns)).Result()
	if err == nil && remaining == 0 {
		s.db.SRem(ctx, nsIndexKey, ns)
		s.db.Del(ctx, schemaIndexKey(ns))
	}

	s.mu.Lock()
	if nsSchemas, ok := s.schemas[ns]; ok {
		delete(nsSchemas, name)
		if len(nsSchemas) == 0 {
			delete(s.schemas, ns)
		}
	}
	s.mu.Unlock()

	return nil
}

func validateSchemaURI(uri *core.URI, def *schema.Def) error {
	if def.NS == nil {
		return errors.New("schema namespace is required")
	}
	if !def.NS.Equals(uri.NS()) {
		return errors.New("schema NS does not match URI NS")
	}
	if def.Name != uri.Schema().String() {
		return errors.New("schema name does not match URI schema")
	}
	return nil
}

func validateFieldChanges(existing, updated *schema.Def) error {
	existingFields := make(map[string]*schema.FieldDef)
	for _, field := range existing.Fields {
		existingFields[field.Name] = field
	}

	for _, field := range updated.Fields {
		if existingField, ok := existingFields[field.Name]; ok {
			if !existingField.Type.Equals(field.Type) {
				return store.ErrFieldChangeType
			}
		}
	}

	return nil
}
