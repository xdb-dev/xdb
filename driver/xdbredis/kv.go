package xdbredis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
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
func (kv *KVStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, error) {
	tx := kv.getTx(ctx)

	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		cmd := tx.HGet(ctx, makeHashKey(key), key.Attr())
		cmds = append(cmds, cmd)
	}

	_, err := tx.Exec(ctx)
	if err != nil {
		return nil, err
	}

	tuples := make([]*types.Tuple, 0, len(keys))

	for i, key := range keys {
		cmd := cmds[i]
		if cmd.Err() != nil {
			return nil, cmd.Err()
		}

		val, err := xdbkv.DecodeValue([]byte(cmd.Val()))
		if err != nil {
			return nil, err
		}

		tuple := types.NewTuple(
			key.Kind(),
			key.ID(),
			key.Attr(),
			val,
		)
		tuples = append(tuples, tuple)
	}

	return tuples, nil
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
	return nil, nil, nil
}

// PutRecords puts records into the key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*types.Record) error {
	return nil
}

// DeleteRecords deletes records from the key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	return nil
}

func (kv *KVStore) getTx(_ context.Context) redis.Pipeliner {
	return kv.db.TxPipeline()
}

func makeHashKey(key interface {
	Kind() string
	ID() string
}) string {
	return fmt.Sprintf("%s:%s", key.Kind(), key.ID())
}

func encodeValue(value any) (string, error) {
	b, err := msgpack.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func decodeValue(value string) (any, error) {
	var v any
	err := msgpack.Unmarshal([]byte(value), &v)
	if err != nil {
		return nil, err
	}

	return v, nil
}
