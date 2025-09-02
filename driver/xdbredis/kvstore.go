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
	_ driver.TupleReader = (*KVStore)(nil)
	_ driver.TupleWriter = (*KVStore)(nil)
)

// KVStore store tuples and records as key-value pairs in Redis.
type KVStore struct {
	db *redis.Client
}

// NewKVStore creates a new KVStore.
func NewKVStore(c *redis.Client) *KVStore {
	return &KVStore{db: c}
}

// GetTuples gets tuples from the key-value store.
func (kv *KVStore) GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error) {
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
func (kv *KVStore) PutTuples(ctx context.Context, tuples []*types.Tuple) error {
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
func (kv *KVStore) DeleteTuples(ctx context.Context, keys []*types.Key) error {
	tx := kv.getTx()

	for _, key := range keys {
		tx.Del(ctx, string(xdbkv.EncodeKey(key)))
	}

	_, err := tx.Exec(ctx)

	return err
}

// GetRecords fetches records from the key-value store.
func (kv *KVStore) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	redisTupleKeys := make([]string, 0, len(keys))

	for _, key := range keys {
		tupleKeys, err := kv.scanTupleKeys(ctx, key)
		if err != nil {
			return nil, nil, err
		}

		redisTupleKeys = append(redisTupleKeys, tupleKeys...)
	}

	tx := kv.getTx()

	cmds := make([]*redis.StringCmd, 0, len(redisTupleKeys))
	for _, key := range redisTupleKeys {
		cmd := tx.Get(ctx, key)
		cmds = append(cmds, cmd)
	}

	_, err := tx.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	recordsMap := make(map[string]*types.Record, len(keys))

	for i, key := range redisTupleKeys {
		cmd := cmds[i]

		err := cmd.Err()
		if err != nil {
			return nil, nil, err
		}

		tupleKey, err := xdbkv.DecodeKey([]byte(key))
		if err != nil {
			return nil, nil, err
		}

		tupleValue, err := xdbkv.DecodeValue([]byte(cmd.Val()))
		if err != nil {
			return nil, nil, err
		}

		parts := tupleKey.Unwrap()
		attr := parts[len(parts)-1]
		recordKey := types.NewKey(parts[:len(parts)-1]...)

		if _, ok := recordsMap[recordKey.String()]; !ok {
			recordsMap[recordKey.String()] = types.NewRecord(recordKey)
		}

		recordsMap[recordKey.String()].Set(attr, tupleValue)
	}

	records := make([]*types.Record, 0, len(recordsMap))
	missing := make([]*types.Key, 0, len(keys))

	for _, key := range keys {
		record, ok := recordsMap[key.String()]
		if !ok {
			missing = append(missing, key)
			continue
		}

		records = append(records, record)
	}

	return records, missing, nil
}

// PutRecords puts records in the key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*types.Record) error {
	allTuples := make([]*types.Tuple, 0)

	for _, record := range records {
		allTuples = append(allTuples, record.Tuples()...)
	}

	return kv.PutTuples(ctx, allTuples)
}

// DeleteRecords deletes records from the key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	tx := kv.getTx()

	for _, key := range keys {
		flatkey := getRecordKey(key)
		tx.Del(ctx, flatkey)
	}

	_, err := tx.Exec(ctx)

	return err
}

func (kv *KVStore) scanTupleKeys(ctx context.Context, key *types.Key) ([]string, error) {
	prefix := string(xdbkv.EncodeKey(key))
	cursor := uint64(0)
	tupleKeys := make([]string, 0)

	for {
		keys, cursor, err := kv.db.Scan(ctx, cursor, prefix+":*", 1000).Result()
		if err != nil {
			return nil, err
		}

		tupleKeys = append(tupleKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	return tupleKeys, nil
}

func (kv *KVStore) getTx() redis.Pipeliner {
	return kv.db.Pipeline()
}
