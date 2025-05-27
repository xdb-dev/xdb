package xdbredis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/encoding/xdbkv"
	"github.com/xdb-dev/xdb/types"
)

var (
	_ driver.TupleReader  = (*KVStore)(nil)
	_ driver.TupleWriter  = (*KVStore)(nil)
	_ driver.RecordReader = (*KVStore)(nil)
	_ driver.RecordWriter = (*KVStore)(nil)
)

// KVStore is a key-value store for Redis.
// It stores tuples in Redis hash maps.
type KVStore struct {
	db *redis.Client
}

// New creates a new Redis driver
func New(c *redis.Client) *KVStore {
	return &KVStore{db: c}
}

// GetTuples gets tuples from the key-value store.
func (kv *KVStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
	tx := kv.getTx(ctx)

	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		cmd := tx.HGet(ctx, makeHashKey(key), key.Attr())
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

		tuple := types.NewTuple(
			key.Kind(),
			key.ID(),
			key.Attr(),
			val,
		)
		tuples = append(tuples, tuple)
	}

	return tuples, missing, nil
}

// PutTuples puts tuples into the key-value store.
func (kv *KVStore) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
	tx := kv.getTx(ctx)

	for _, tuple := range tuples {
		hmkey := makeHashKey(tuple)
		hmval, err := xdbkv.EncodeValue(tuple.Value())
		if err != nil {
			return err
		}

		tx.HSet(ctx, hmkey, tuple.Attr(), hmval)
	}

	_, err := tx.Exec(ctx)

	return err
}

// DeleteTuples deletes tuples from the key-value store.
func (kv *KVStore) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	tx := kv.getTx(ctx)

	for _, key := range keys {
		tx.HDel(ctx, makeHashKey(key), key.Attr())
	}

	_, err := tx.Exec(ctx)
	return err
}

// GetRecords gets records from the key-value store.
func (kv *KVStore) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	tx := kv.getTx(ctx)

	cmds := make([]*redis.MapStringStringCmd, 0, len(keys))
	for _, key := range keys {
		cmd := tx.HGetAll(ctx, makeHashKey(key))
		cmds = append(cmds, cmd)
	}

	_, err := tx.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	records := make([]*types.Record, 0, len(keys))
	missing := make([]*types.Key, 0, len(keys))

	for i, key := range keys {
		cmd := cmds[i]

		err := cmd.Err()
		if err != nil {
			return nil, nil, err
		}

		attrs := cmd.Val()
		if len(attrs) == 0 {
			missing = append(missing, key)
			continue
		}

		record := types.NewRecord(
			key.Kind(),
			key.ID(),
		)

		for attr, val := range attrs {
			vv, err := xdbkv.DecodeValue([]byte(val))
			if err != nil {
				return nil, nil, err
			}

			record.Set(attr, vv)
		}

		records = append(records, record)
	}

	return records, missing, nil
}

// PutRecords puts records into the key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*types.Record) error {
	tuples := make([]*types.Tuple, 0, len(records))

	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}

	return kv.PutTuples(ctx, tuples)
}

// DeleteRecords deletes records from the key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	tx := kv.getTx(ctx)

	for _, key := range keys {
		tx.Del(ctx, makeHashKey(key))
	}

	_, err := tx.Exec(ctx)

	return err
}

func (kv *KVStore) getTx(_ context.Context) redis.Pipeliner {
	return kv.db.Pipeline()
}

func makeHashKey(key interface {
	Kind() string
	ID() string
}) string {
	return fmt.Sprintf("%s:%s", key.Kind(), key.ID())
}
