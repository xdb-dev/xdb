// Package xdbredis provides a Redis-backed driver for XDB.
package xdbredis

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/codec"
	"github.com/xdb-dev/xdb/codec/msgpack"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
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
	db    *redis.Client
	codec codec.KVCodec
}

// New creates a new Redis driver
func New(c *redis.Client) *KVStore {
	return &KVStore{db: c, codec: msgpack.New()}
}

// GetTuples gets tuples from the key-value store.
func (kv *KVStore) GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error) {
	tx := kv.getTx(ctx)

	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		hmid, hmattr, err := kv.codec.EncodeKey(key)
		if err != nil {
			return nil, nil, err
		}

		cmd := tx.HGet(ctx, string(hmid), string(hmattr))
		cmds = append(cmds, cmd)
	}

	_, err := tx.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	tuples := make([]*core.Tuple, 0, len(keys))
	missing := make([]*core.Key, 0, len(keys))

	for i, key := range keys {
		cmd := cmds[i]

		err := cmd.Err()
		if errors.Is(err, redis.Nil) {
			missing = append(missing, key)
			continue
		} else if err != nil {
			return nil, nil, err
		}

		val, err := kv.codec.DecodeValue([]byte(cmd.Val()))
		if err != nil {
			return nil, nil, err
		}

		tuple := core.NewTuple(key.ID(), key.Attr(), val)
		tuples = append(tuples, tuple)
	}

	return tuples, missing, nil
}

// PutTuples puts tuples into the key-value store.
func (kv *KVStore) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	tx := kv.getTx(ctx)

	for _, tuple := range tuples {
		hmid, hmattr, err := kv.codec.EncodeKey(tuple.Key())
		if err != nil {
			return err
		}

		hmval, err := kv.codec.EncodeValue(tuple.Value())
		if err != nil {
			return err
		}

		tx.HSet(ctx, string(hmid), string(hmattr), hmval)
	}

	_, err := tx.Exec(ctx)

	return err
}

// DeleteTuples deletes tuples from the key-value store.
func (kv *KVStore) DeleteTuples(ctx context.Context, keys []*core.Key) error {
	tx := kv.getTx(ctx)

	for _, key := range keys {
		hmid, hmattr, err := kv.codec.EncodeKey(key)
		if err != nil {
			return err
		}
		tx.HDel(ctx, string(hmid), string(hmattr))
	}

	_, err := tx.Exec(ctx)
	return err
}

// GetRecords gets records from the key-value store.
func (kv *KVStore) GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error) {
	tx := kv.getTx(ctx)

	cmds := make([]*redis.MapStringStringCmd, 0, len(keys))
	for _, key := range keys {
		hmid, _, err := kv.codec.EncodeKey(key)
		if err != nil {
			return nil, nil, err
		}
		cmd := tx.HGetAll(ctx, string(hmid))
		cmds = append(cmds, cmd)
	}

	_, err := tx.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, nil, err
	}

	records := make([]*core.Record, 0, len(keys))
	missing := make([]*core.Key, 0, len(keys))

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

		record := core.NewRecord(key.ID()...)

		for attr, val := range attrs {
			decodedAttr, err := kv.codec.DecodeAttr([]byte(attr))
			if err != nil {
				return nil, nil, err
			}

			decodedVal, err := kv.codec.DecodeValue([]byte(val))
			if err != nil {
				return nil, nil, err
			}

			record.Set(decodedAttr, decodedVal)
		}

		records = append(records, record)
	}

	return records, missing, nil
}

// PutRecords puts records into the key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*core.Record) error {
	tuples := make([]*core.Tuple, 0, len(records))

	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}

	return kv.PutTuples(ctx, tuples)
}

// DeleteRecords deletes records from the key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*core.Key) error {
	tx := kv.getTx(ctx)

	for _, key := range keys {
		hmid, _, err := kv.codec.EncodeKey(key)
		if err != nil {
			return err
		}
		tx.Del(ctx, string(hmid))
	}

	_, err := tx.Exec(ctx)

	return err
}

func (kv *KVStore) getTx(_ context.Context) redis.Pipeliner {
	return kv.db.Pipeline()
}
