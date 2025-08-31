// Package xdbredis provides a Redis-backed driver for XDB.
package xdbredis

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/encoding/xdbkv"
	"github.com/xdb-dev/xdb/types"
)

var (
	_ driver.TupleReader = (*TupleKVStore)(nil)
	_ driver.TupleWriter = (*TupleKVStore)(nil)
)

// TupleKVStore manages tuple storage in Redis.
type TupleKVStore struct {
	db *redis.Client
}

// NewTupleKVStore creates a new TupleKVStore.
func NewTupleKVStore(c *redis.Client) *TupleKVStore {
	return &TupleKVStore{db: c}
}

// GetTuples gets tuples from the key-value store.
func (kv *TupleKVStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	tx := kv.getTx()

	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		encodedKey := xdbkv.EncodeKey(key)
		cmd := tx.Get(ctx, string(encodedKey))

		cmds = append(cmds, cmd)
	}

	_, err := tx.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	tuples := make([]*types.Tuple, 0, len(keys))
	missing := make([]*types.Key, 0, len(keys))

	for i, key := range keys {
		cmd := cmds[i]

		err := cmd.Err()
		if errors.Is(err, redis.Nil) {
			missing = append(missing, key)
			continue
		} else if err != nil {
			return nil, nil, err
		}

		val, err := xdbkv.DecodeValue([]byte(cmd.Val()))
		if err != nil {
			return nil, nil, err
		}

		tuple := types.NewTuple(key, val)

		tuples = append(tuples, tuple)
	}

	return tuples, missing, nil
}

// PutTuples puts tuples into the key-value store.
func (kv *TupleKVStore) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
	tx := kv.getTx()

	for _, tuple := range tuples {
		key, value, err := xdbkv.EncodeTuple(tuple)
		if err != nil {
			return err
		}

		tx.Set(ctx, string(key), string(value), 0)
	}

	_, err := tx.Exec(ctx)

	return err
}

// DeleteTuples deletes tuples from the key-value store.
func (kv *TupleKVStore) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	tx := kv.getTx()

	for _, key := range keys {
		tx.Del(ctx, string(xdbkv.EncodeKey(key)))
	}

	_, err := tx.Exec(ctx)

	return err
}

func (kv *TupleKVStore) getTx() redis.Pipeliner {
	return kv.db.Pipeline()
}
