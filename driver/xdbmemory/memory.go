// Package xdbmemory provides an in-memory driver implementation for XDB.
package xdbmemory

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/xdb-dev/xdb/types"
)

// MemoryDriver is an in-memory driver for XDB.
type MemoryDriver struct {
	mu     sync.RWMutex
	tuples map[string]*types.Tuple
}

// New creates a new in-memory driver.
func New() *MemoryDriver {
	return &MemoryDriver{
		tuples: make(map[string]*types.Tuple),
	}
}

// GetTuples returns the tuples for the given keys.
func (d *MemoryDriver) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tuples := make([]*types.Tuple, 0, len(keys))
	missed := make([]*types.Key, 0, len(keys))

	for _, key := range keys {
		tuple, ok := d.tuples[encodeKey(key)]
		if !ok {
			missed = append(missed, key)
			continue
		}

		tuples = append(tuples, tuple)
	}

	return tuples, missed, nil
}

// PutTuples saves the tuples.
func (d *MemoryDriver) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, tuple := range tuples {
		d.tuples[encodeKey(tuple.Key())] = tuple
	}

	return nil
}

// DeleteTuples deletes the tuples for the given keys.
func (d *MemoryDriver) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		delete(d.tuples, encodeKey(key))
	}

	return nil
}

// GetRecords returns the records for the given keys.
func (d *MemoryDriver) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	records := make([]*types.Record, 0, len(keys))
	missed := make([]*types.Key, 0, len(keys))

	for _, key := range keys {
		record := types.NewRecord(key.Kind(), key.ID())

		for k, t := range d.tuples {
			if !strings.HasPrefix(k, encodeKey(key)) {
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
func (d *MemoryDriver) PutRecords(ctx context.Context, records []*types.Record) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, record := range records {
		for _, tuple := range record.Tuples() {
			d.tuples[encodeKey(tuple.Key())] = tuple
		}
	}

	return nil
}

// DeleteRecords deletes the records for the given keys.
func (d *MemoryDriver) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		for k := range d.tuples {
			if !strings.HasPrefix(k, encodeKey(key)) {
				continue
			}

			delete(d.tuples, k)
		}
	}

	return nil
}

func encodeKey(key *types.Key) string {
	if key.Attr() != "" {
		return fmt.Sprintf("%s/%s/%s", key.Kind(), key.ID(), key.Attr())
	}

	return fmt.Sprintf("%s/%s", key.Kind(), key.ID())
}
