// Package xdbmemory provides an in-memory driver implementation for XDB.
package xdbmemory

import (
	"context"
	"sync"

	"github.com/xdb-dev/xdb/core"
)

// MemoryDriver is an in-memory driver for XDB.
type MemoryDriver struct {
	mu          sync.RWMutex
	repos       map[string]*core.Repo
	collections map[string]map[string]*core.Collection
	tuples      map[string]map[string]*core.Tuple
}

// New creates a new in-memory driver.
func New() *MemoryDriver {
	return &MemoryDriver{
		tuples: make(map[string]map[string]*core.Tuple),
	}
}

// CreateRepo creates a new repository.
func (d *MemoryDriver) CreateRepo(ctx context.Context, repo *core.Repo) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.repos[repo.String()] = repo

	return nil
}

// CreateSchema creates a new schema.
func (d *MemoryDriver) CreateSchema(ctx context.Context, coll *core.Collection) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.collections[coll.Repo][coll.ID] = coll

	return nil
}

// GetTuples returns the tuples for the given keys.
func (d *MemoryDriver) GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tuples := make([]*core.Tuple, 0, len(keys))
	missed := make([]*core.Key, 0, len(keys))

	for _, key := range keys {
		id := key.ID().String()
		attr := key.Attr().String()

		if _, ok := d.tuples[id]; !ok {
			missed = append(missed, key)
			continue
		}

		tuple, ok := d.tuples[id][attr]
		if !ok {
			missed = append(missed, key)
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

// DeleteTuples deletes the tuples for the given keys.
func (d *MemoryDriver) DeleteTuples(ctx context.Context, keys []*core.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		id := key.ID().String()
		attr := key.Attr().String()

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

// GetRecords returns the records for the given keys.
func (d *MemoryDriver) GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	records := make([]*core.Record, 0, len(keys))
	missed := make([]*core.Key, 0, len(keys))

	for _, key := range keys {
		record := core.NewRecord(key.ID()...)

		id := key.ID().String()

		if _, ok := d.tuples[id]; !ok {
			missed = append(missed, key)
			continue
		}

		for _, t := range d.tuples[id] {
			record.Set(t.Attr(), t.Value())
		}

		if record.IsEmpty() {
			missed = append(missed, key)
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

// DeleteRecords deletes the records for the given keys.
func (d *MemoryDriver) DeleteRecords(ctx context.Context, keys []*core.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		id := key.ID().String()

		if _, ok := d.tuples[id]; !ok {
			continue
		}

		delete(d.tuples, id)
	}

	return nil
}
