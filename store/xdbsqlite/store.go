package xdbsqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// Store is a SQLite-backed implementation of [store.Store].
type Store struct {
	db    *sql.DB
	cache sync.Map // ns/schema -> *schema.Def
}

// Option configures a [Store].
type Option func(*Store)

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// New creates a new SQLite store backed by the given [*sql.DB].
// The caller is responsible for opening and closing the database connection.
func New(db *sql.DB, opts ...Option) (*Store, error) {
	s := &Store{db: db}
	for _, opt := range opts {
		opt(s)
	}

	if err := xsql.NewQueries(s.db).Bootstrap(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

// cacheKey returns the schema cache key for a URI.
func cacheKey(uri *core.URI) string {
	return uri.NS().String() + "/" + uri.Schema().String()
}

// cachedSchema returns the schema definition for the given URI,
// using the in-memory cache when possible. Returns nil if no schema exists.
func (s *Store) cachedSchema(ctx context.Context, q *xsql.Queries, uri *core.URI) (*schema.Def, error) {
	key := cacheKey(uri)

	if v, ok := s.cache.Load(key); ok {
		return v.(*schema.Def), nil
	}

	data, err := q.GetSchema(ctx, xsql.GetSchemaParams{
		Namespace: uri.NS().String(),
		Schema:    uri.Schema().String(),
	})
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var def schema.Def
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}

	s.cache.Store(key, &def)
	return &def, nil
}

// invalidateSchema removes a schema from the cache.
func (s *Store) invalidateSchema(uri *core.URI) {
	s.cache.Delete(cacheKey(uri))
}
