package xdbredis

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/codec/msgpack"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

var (
	_ store.HealthChecker = (*Store)(nil)
	_ store.SchemaStore   = (*Store)(nil)
	_ store.SchemaReader  = (*Store)(nil)
	_ store.RecordStore   = (*Store)(nil)
	_ store.TupleStore    = (*Store)(nil)
)

// Store is a Redis-backed implementation of the XDB store interfaces.
type Store struct {
	db    *redis.Client
	codec codec.KVCodec

	mu      sync.RWMutex
	schemas map[string]map[string]*schema.Def
}

// NewStore creates a new Redis Store that implements all store interfaces.
func NewStore(c *redis.Client) (*Store, error) {
	s := &Store{
		db:    c,
		codec: msgpack.New(),
	}

	if err := s.loadSchemas(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

// Close closes the underlying Redis connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Health checks if the Redis connection is alive.
func (s *Store) Health(ctx context.Context) error {
	return s.db.Ping(ctx).Err()
}
