package xdbredis

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/core"
)

const defaultPrefix = "xdb"

// Store is a Redis-backed implementation of [store.Store].
type Store struct {
	client redis.UniversalClient
	prefix string
}

// Option configures a [Store].
type Option func(*Store)

// WithPrefix sets the key prefix. Default: "xdb".
func WithPrefix(prefix string) Option {
	return func(s *Store) {
		s.prefix = prefix
	}
}

// New creates a new Redis store.
func New(client redis.UniversalClient, opts ...Option) *Store {
	s := &Store{
		client: client,
		prefix: defaultPrefix,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// --- Key helpers ---

// recordKey returns the Redis key for a record: {prefix}:{ns}:{schema}:{id}.
func (s *Store) recordKey(uri *core.URI) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		s.prefix,
		uri.NS().String(),
		uri.Schema().String(),
		uri.ID().String(),
	)
}

// schemaKey returns the Redis key for a schema: {prefix}:{ns}:{schema}:_schema.
func (s *Store) schemaKey(uri *core.URI) string {
	return fmt.Sprintf("%s:%s:%s:_schema",
		s.prefix,
		uri.NS().String(),
		uri.Schema().String(),
	)
}

// recordIndexKey returns the key for the record index set: {prefix}:{ns}:{schema}:_idx.
func (s *Store) recordIndexKey(uri *core.URI) string {
	return fmt.Sprintf("%s:%s:%s:_idx",
		s.prefix,
		uri.NS().String(),
		uri.Schema().String(),
	)
}

// schemaIndexKey returns the key for the schema index set: {prefix}:{ns}:_idx.
func (s *Store) schemaIndexKey(uri *core.URI) string {
	return fmt.Sprintf("%s:%s:_idx", s.prefix, uri.NS().String())
}

// nsIndexKey returns the key for the namespace index set: {prefix}:_idx.
func (s *Store) nsIndexKey() string {
	return fmt.Sprintf("%s:_idx", s.prefix)
}
