package xdbsqlite

import (
	"context"
	"database/sql"
	"slices"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

func (s *Store) loadSchemas(ctx context.Context) error {
	s.mu.Lock()
	s.schemas = make(map[string]map[string]*schema.Def)
	s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	driver := NewSchemaStoreTx(tx)

	namespaces, err := driver.ListNamespaces(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	for _, ns := range namespaces {
		uri, err := core.ParseURI("xdb://" + ns.String())
		if err != nil {
			return err
		}

		schemas, err := driver.ListSchemas(ctx, uri)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return err
		}

		s.mu.Lock()
		s.schemas[ns.String()] = make(map[string]*schema.Def)
		for _, def := range schemas {
			if def.NS == nil || !def.NS.Equals(ns) {
				s.mu.Unlock()
				return errors.New("schema NS does not match namespace")
			}
			s.schemas[ns.String()][def.Name] = def
		}
		s.mu.Unlock()
	}

	return nil
}

// GetSchema returns the schema definition for the given URI.
func (s *Store) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	if def, err := s.getSchemaFromCache(uri); err == nil {
		return def, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	return s.getSchemaWithTx(ctx, tx, uri)
}

func (s *Store) getSchemaFromCache(uri *core.URI) (*schema.Def, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns := uri.NS().String()
	schemaKey := uri.Schema().String()

	if nsSchemas, ok := s.schemas[ns]; ok {
		if def, ok := nsSchemas[schemaKey]; ok {
			return def, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *Store) getSchemaWithTx(ctx context.Context, tx *sql.Tx, uri *core.URI) (*schema.Def, error) {
	ns := uri.NS().String()
	schemaKey := uri.Schema().String()

	driverTx := NewSchemaStoreTx(tx)
	def, err := driverTx.GetSchema(ctx, uri)
	if err != nil {
		return nil, err
	}

	if def.NS == nil || !def.NS.Equals(uri.NS()) {
		return nil, errors.New("schema NS does not match URI NS")
	}
	if def.Name != schemaKey {
		return nil, errors.New("schema name does not match URI schema")
	}

	s.mu.Lock()
	if _, ok := s.schemas[ns]; !ok {
		s.schemas[ns] = make(map[string]*schema.Def)
	}
	s.schemas[ns][schemaKey] = def
	s.mu.Unlock()

	return def, nil
}

// ListNamespaces returns all unique namespaces that contain schemas.
func (s *Store) ListNamespaces(ctx context.Context) ([]*core.NS, error) {
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
func (s *Store) ListSchemas(ctx context.Context, uri *core.URI) ([]*schema.Def, error) {
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	driverTx := NewSchemaStoreTx(tx)
	if err := driverTx.PutSchema(ctx, uri, def); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	ns := uri.NS().String()
	schemaKey := uri.Schema().String()

	s.mu.Lock()
	if s.schemas[ns] == nil {
		s.schemas[ns] = make(map[string]*schema.Def)
	}
	s.schemas[ns][schemaKey] = def
	s.mu.Unlock()

	return nil
}

// DeleteSchema removes a schema definition.
func (s *Store) DeleteSchema(ctx context.Context, uri *core.URI) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	driverTx := NewSchemaStoreTx(tx)
	if err := driverTx.DeleteSchema(ctx, uri); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	ns := uri.NS().String()
	schemaKey := uri.Schema().String()

	s.mu.Lock()
	if nsSchemas, ok := s.schemas[ns]; ok {
		delete(nsSchemas, schemaKey)

		if len(nsSchemas) == 0 {
			delete(s.schemas, ns)
		}
	}
	s.mu.Unlock()

	return nil
}