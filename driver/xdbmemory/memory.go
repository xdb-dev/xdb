// Package xdbmemory provides an in-memory driver implementation for XDB.
package xdbmemory

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/schema"
)

// Config holds the configuration for the in-memory driver.
type Config struct {
	Enabled bool `env:"ENABLED"`
}

// MemoryDriver is an in-memory driver for XDB.
type MemoryDriver struct {
	mu      sync.RWMutex
	tuples  map[string]map[string]*core.Tuple
	schemas map[string]map[string]*schema.Def
}

// New creates a new in-memory driver.
func New() *MemoryDriver {
	return &MemoryDriver{
		tuples:  make(map[string]map[string]*core.Tuple),
		schemas: make(map[string]map[string]*schema.Def),
	}
}

// GetSchema returns the schema definition for the given URI.
func (d *MemoryDriver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	schemas, ok := d.schemas[uri.NS().String()]
	if !ok {
		return nil, driver.ErrNotFound
	}

	s, ok := schemas[uri.Schema().String()]
	if !ok {
		return nil, driver.ErrNotFound
	}

	return s, nil
}

// ListSchemas returns all schema definitions in the given namespace.
func (d *MemoryDriver) ListSchemas(ctx context.Context, uri *core.URI) ([]*schema.Def, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ns := uri.NS().String()
	if _, ok := d.schemas[ns]; !ok {
		return nil, nil
	}

	schemas := slices.Collect(maps.Values(d.schemas[ns]))
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

// PutSchema saves the schema definition.
func (d *MemoryDriver) PutSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ns := uri.NS().String()
	if _, ok := d.schemas[ns]; !ok {
		d.schemas[ns] = make(map[string]*schema.Def)
	}

	schemaKey := uri.Schema().String()
	existing, exists := d.schemas[ns][schemaKey]
	if exists {
		if existing.Mode != def.Mode {
			return driver.ErrSchemaModeChanged
		}

		for _, newField := range def.Fields {
			if oldField := existing.GetField(newField.Name); oldField != nil {
				if !oldField.Type.Equals(newField.Type) {
					return driver.ErrFieldChangeType
				}
			}
		}
	}

	d.schemas[ns][schemaKey] = def

	return nil
}

// DeleteSchema deletes the schema definition.
func (d *MemoryDriver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	schemas, ok := d.schemas[uri.NS().String()]
	if !ok {
		return nil
	}

	delete(schemas, uri.Schema().String())
	if len(schemas) == 0 {
		delete(d.schemas, uri.NS().String())
	}

	return nil
}

// GetTuples returns the tuples for the given uris.
func (d *MemoryDriver) GetTuples(ctx context.Context, uris []*core.URI) ([]*core.Tuple, []*core.URI, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tuples := make([]*core.Tuple, 0, len(uris))
	missed := make([]*core.URI, 0, len(uris))

	for _, uri := range uris {
		id := uri.ID().String()
		attr := uri.Attr().String()

		if _, ok := d.tuples[id]; !ok {
			missed = append(missed, uri)
			continue
		}

		tuple, ok := d.tuples[id][attr]
		if !ok {
			missed = append(missed, uri)
			continue
		}

		tuples = append(tuples, tuple)
	}

	return tuples, missed, nil
}

// PutTuples saves the tuples.
func (d *MemoryDriver) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, tuple := range tuples {
		id := tuple.ID().String()
		attr := tuple.Attr().String()

		if _, ok := d.tuples[id]; !ok {
			d.tuples[id] = make(map[string]*core.Tuple)
		}

		d.tuples[id][attr] = tuple
	}

	return nil
}

// DeleteTuples deletes the tuples for the given uris.
func (d *MemoryDriver) DeleteTuples(ctx context.Context, uris []*core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, uri := range uris {
		id := uri.ID().String()
		attr := uri.Attr().String()

		if _, ok := d.tuples[id]; !ok {
			continue
		}

		if _, ok := d.tuples[id][attr]; !ok {
			continue
		}

		delete(d.tuples[id], attr)
	}

	return nil
}

// GetRecords returns the records for the given uris.
func (d *MemoryDriver) GetRecords(ctx context.Context, uris []*core.URI) ([]*core.Record, []*core.URI, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	records := make([]*core.Record, 0, len(uris))
	missed := make([]*core.URI, 0, len(uris))

	for _, uri := range uris {
		record := core.NewRecord(uri.NS().String(), uri.Schema().String(), uri.ID().String())

		id := uri.ID().String()

		if _, ok := d.tuples[id]; !ok {
			missed = append(missed, uri)
			continue
		}

		for _, t := range d.tuples[id] {
			record.Set(t.Attr().String(), t.Value())
		}

		if record.IsEmpty() {
			missed = append(missed, uri)
			continue
		}

		records = append(records, record)
	}

	return records, missed, nil
}

// PutRecords saves the records.
func (d *MemoryDriver) PutRecords(ctx context.Context, records []*core.Record) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, record := range records {
		id := record.ID().String()

		if _, ok := d.tuples[id]; !ok {
			d.tuples[id] = make(map[string]*core.Tuple)
		}

		for _, tuple := range record.Tuples() {
			attr := tuple.Attr().String()

			d.tuples[id][attr] = tuple
		}
	}

	return nil
}

// DeleteRecords deletes the records for the given uris.
func (d *MemoryDriver) DeleteRecords(ctx context.Context, uris []*core.URI) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, uri := range uris {
		id := uri.ID().String()

		if _, ok := d.tuples[id]; !ok {
			continue
		}

		delete(d.tuples, id)
	}

	return nil
}
