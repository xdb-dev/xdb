// Package xdbmemory provides an in-memory driver implementation for XDB.
package xdbmemory

import (
	"context"
	"strings"
	"sync"

	"github.com/xdb-dev/xdb/core"
)

// MemoryDriver is an in-memory driver for XDB.
type MemoryDriver struct {
	mu     sync.RWMutex
	tuples map[string]*core.Tuple
}

// New creates a new in-memory driver.
func New() *MemoryDriver {
	return &MemoryDriver{
		tuples: make(map[string]*core.Tuple),
	}
}

// GetTuples returns the tuples for the given keys.
func (d *MemoryDriver) GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tuples := make([]*core.Tuple, 0, len(keys))
	missed := make([]*core.Key, 0, len(keys))

	for _, key := range keys {
		tuple, ok := d.tuples[key.String()]
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
		d.tuples[tuple.Key().String()] = tuple
	}

	return nil
}

// DeleteTuples deletes the tuples for the given keys.
func (d *MemoryDriver) DeleteTuples(ctx context.Context, keys []*core.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		delete(d.tuples, key.String())
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

		for k, t := range d.tuples {
			if !strings.HasPrefix(k, key.String()) {
				continue
			}

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
		for _, tuple := range record.Tuples() {
			d.tuples[tuple.Key().String()] = tuple
		}
	}

	return nil
}

// DeleteRecords deletes the records for the given keys.
func (d *MemoryDriver) DeleteRecords(ctx context.Context, keys []*core.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		for k := range d.tuples {
			if !strings.HasPrefix(k, key.String()) {
				continue
			}

			delete(d.tuples, k)
		}
	}

	return nil
}
