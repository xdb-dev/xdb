package store

import (
	"context"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// newDefCache creates a caching layer over the driver for
// [schema.Def] reads. It sits beneath the enforcement middleware so
// every validation read hits the cache, and is skipped entirely
// inside transactions (a tx that writes a Def must read its own
// write, not a cached one) — the facade invalidates the cache after
// every transactional write instead.
func newDefCache(next Driver) *cachingDriver {
	return &cachingDriver{
		Driver: next,
		defs:   make(map[string]*schema.Def),
	}
}

// cachingDriver caches GetSchema reads and keeps the cache coherent on
// def writes. Tuple operations and def scans are inherited unchanged
// from the embedded [Driver].
type cachingDriver struct {
	Driver
	defs map[string]*schema.Def
	mu   sync.RWMutex
}

func (c *cachingDriver) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	key := uri.Path()

	c.mu.RLock()
	cached, ok := c.defs[key]
	c.mu.RUnlock()
	if ok {
		return cached, nil
	}

	def, err := c.Driver.GetSchema(ctx, uri)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.defs[key] = def
	c.mu.Unlock()

	return def, nil
}

func (c *cachingDriver) CreateSchema(ctx context.Context, def *schema.Def) error {
	if err := c.Driver.CreateSchema(ctx, def); err != nil {
		return err
	}
	c.store(def)
	return nil
}

func (c *cachingDriver) PutSchema(ctx context.Context, def *schema.Def) error {
	if err := c.Driver.PutSchema(ctx, def); err != nil {
		return err
	}
	c.store(def)
	return nil
}

func (c *cachingDriver) DeleteSchema(ctx context.Context, uri *core.URI) error {
	if err := c.Driver.DeleteSchema(ctx, uri); err != nil {
		return err
	}

	c.mu.Lock()
	delete(c.defs, uri.Path())
	c.mu.Unlock()

	return nil
}

func (c *cachingDriver) store(def *schema.Def) {
	c.mu.Lock()
	c.defs[def.URI.Path()] = def
	c.mu.Unlock()
}

// invalidateAll drops every cached def. Called by the facade after
// transactional writes, which bypass this layer.
func (c *cachingDriver) invalidateAll() {
	c.mu.Lock()
	clear(c.defs)
	c.mu.Unlock()
}
