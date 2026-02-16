package xdbsqlite

import (
	"context"
	"database/sql"
	"slices"
	"sync"

	"github.com/gojekfarm/xtools/errors"
	"github.com/ncruces/go-sqlite3/driver"

	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/x"
)

var (
	_ store.HealthChecker = (*Store)(nil)
	_ store.SchemaStore   = (*Store)(nil)
	_ store.SchemaReader  = (*Store)(nil)
	_ store.RecordStore   = (*Store)(nil)
)

type Store struct {
	cfg Config
	db  *sql.DB

	// Schema cache
	mu      sync.RWMutex
	schemas map[string]map[string]*schema.Def
}

func New(cfg Config) (*Store, error) {
	ctx := context.Background()

	s := &Store{cfg: cfg}

	db, err := s.newDB()
	if err != nil {
		return nil, err
	}

	s.db = db

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	driver := NewSchemaStoreTx(tx)
	if err := driver.Migrate(ctx); err != nil {
		_ = tx.Rollback()
		_ = db.Close()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := s.loadSchemas(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// Health checks if the SQLite database connection is alive.
func (s *Store) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) newDB() (*sql.DB, error) {
	db, err := driver.Open(s.cfg.DSN())
	if err != nil {
		return nil, err
	}

	if s.cfg.InMemory {
		db.SetMaxOpenConns(1)
	}

	return db, nil
}

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
// Reads from in-memory cache first, falls back to database on cache miss.
// Uses SchemaDriverTx for database queries - never queries directly.
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
// Reads from in-memory cache.
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
// Reads from in-memory cache.
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
			records, missed, err = driver.GetRecords(ctx, schemaURI, ids)
			if err != nil {
				return nil, nil, err
			}
		case schema.ModeStrict, schema.ModeDynamic:
			driver := NewSQLDriverTx(tx, schemaDef)
			records, missed, err = driver.GetRecords(ctx, schemaURI, schemaDef, ids)
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

	if err := tx.Commit(); err != nil {
		return err
	}

	return s.refreshSchemaCache(ctx, records)
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

func (s *Store) refreshSchemaCache(ctx context.Context, records []*core.Record) error {
	seen := make(map[string]bool)
	for _, record := range records {
		key := record.NS().String() + "/" + record.Schema().String()
		if seen[key] {
			continue
		}
		seen[key] = true

		schemaURI := record.SchemaURI()

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		driverTx := NewSchemaStoreTx(tx)
		def, err := driverTx.GetSchema(ctx, schemaURI)
		_ = tx.Rollback()

		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				continue
			}
			return err
		}

		ns := record.NS().String()
		schemaKey := record.Schema().String()
		s.mu.Lock()
		if s.schemas[ns] == nil {
			s.schemas[ns] = make(map[string]*schema.Def)
		}
		s.schemas[ns][schemaKey] = def
		s.mu.Unlock()
	}
	return nil
}
