// Package xdbmemory provides an in-memory driver implementation for XDB.
package xdbmemory

import (
	"context"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
)

// Config holds the configuration for the in-memory driver.
type Config struct {
	Enabled bool `env:"ENABLED"`
}

// MemoryDriver is an in-memory driver for XDB.
type MemoryDriver struct {
	mu     sync.RWMutex
	repos  map[string]*core.Repo
	tuples map[string]map[string]*core.Tuple
}

// New creates a new in-memory driver.
func New() *MemoryDriver {
	return &MemoryDriver{
		repos:  make(map[string]*core.Repo),
		tuples: make(map[string]map[string]*core.Tuple),
	}
}

// GetRepo returns the repo for the given name.
func (d *MemoryDriver) GetRepo(ctx context.Context, name string) (*core.Repo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	repo, ok := d.repos[name]
	if !ok {
		return nil, driver.ErrNotFound
	}

	return repo, nil
}

// ListRepos returns the list of repos.
func (d *MemoryDriver) ListRepos(ctx context.Context) ([]*core.Repo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	repos := make([]*core.Repo, 0, len(d.repos))
	for _, repo := range d.repos {
		repos = append(repos, repo)
	}

	return repos, nil
}

// MakeRepo saves the repo.
func (d *MemoryDriver) MakeRepo(ctx context.Context, repo *core.Repo) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.repos[repo.Name()] = repo
	d.tuples[repo.Name()] = make(map[string]*core.Tuple)

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
		repo := uri.Repo()
		record := core.NewRecord(repo, uri.ID()...)

		id := uri.ID().String()

		if _, ok := d.tuples[id]; !ok {
			missed = append(missed, uri)
			continue
		}

		for _, t := range d.tuples[id] {
			record.Set(t.Attr(), t.Value())
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
