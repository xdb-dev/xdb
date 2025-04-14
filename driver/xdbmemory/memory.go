package xdbmemory

import (
	"context"
	"strings"
	"sync"

	"github.com/xdb-dev/xdb/types"
)

// MemoryDriver is an in-memory driver for XDB.
type MemoryDriver struct {
	mu     sync.RWMutex
	tuples map[string]*types.Tuple
	edges  map[string]*types.Edge
}

// New creates a new in-memory driver.
func New() *MemoryDriver {
	return &MemoryDriver{
		tuples: make(map[string]*types.Tuple),
		edges:  make(map[string]*types.Edge),
	}
}

// GetTuples returns the tuples for the given keys.
func (d *MemoryDriver) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	tuples := make([]*types.Tuple, 0, len(keys))
	missed := make([]*types.Key, 0, len(keys))

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
func (d *MemoryDriver) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, tuple := range tuples {
		d.tuples[tuple.Key().String()] = tuple
	}

	return nil
}

// DeleteTuples deletes the tuples for the given keys.
func (d *MemoryDriver) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		delete(d.tuples, key.String())
	}

	return nil
}

// GetEdges returns the edges for the given keys.
func (d *MemoryDriver) GetEdges(ctx context.Context, keys []*types.Key) ([]*types.Edge, []*types.Key, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	edges := make([]*types.Edge, 0, len(keys))
	missed := make([]*types.Key, 0, len(keys))

	for _, key := range keys {
		edge, ok := d.edges[key.String()]
		if !ok {
			missed = append(missed, key)
			continue
		}

		edges = append(edges, edge)
	}

	return edges, missed, nil
}

// PutEdges saves the edges.
func (d *MemoryDriver) PutEdges(ctx context.Context, edges []*types.Edge) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, edge := range edges {
		d.edges[edge.Key().String()] = edge
	}

	return nil
}

// DeleteEdges deletes the edges for the given keys.
func (d *MemoryDriver) DeleteEdges(ctx context.Context, keys []*types.Key) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, key := range keys {
		delete(d.edges, key.String())
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
			if !strings.HasPrefix(k, key.String()) {
				continue
			}

			record.Set(t.Attr(), t.Value())
		}

		for k, e := range d.edges {
			if !strings.HasPrefix(k, key.String()) {
				continue
			}

			record.AddEdge(e.Name(), e.Target())
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
			d.tuples[tuple.Key().String()] = tuple
		}

		for _, edge := range record.Edges() {
			d.edges[edge.Key().String()] = edge
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
			if !strings.HasPrefix(k, key.String()) {
				continue
			}

			delete(d.tuples, k)
		}

		for k := range d.edges {
			if !strings.HasPrefix(k, key.String()) {
				continue
			}

			delete(d.edges, k)
		}
	}

	return nil
}
