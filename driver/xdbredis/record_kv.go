// Package xdbredis provides a Redis-backed driver for XDB.
package xdbredis

import (
	"context"
	"errors"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/encoding/xdbkv"
	"github.com/xdb-dev/xdb/types"
)

var (
	_ driver.RecordReader = (*RecordKVStore)(nil)
	_ driver.RecordWriter = (*RecordKVStore)(nil)
)

// RecordKVStore manages record storage in Redis.
type RecordKVStore struct {
	db *redis.Client
}

// NewRecordKVStore creates a new RecordKVStore.
func NewRecordKVStore(c *redis.Client) *RecordKVStore {
	return &RecordKVStore{db: c}
}

// GetRecords gets records from Redis.
func (kv *RecordKVStore) GetRecords(ctx context.Context, keys []*types.Key) ([]*types.Record, []*types.Key, error) {
	tx := kv.getTx()

	cmds := make([]*redis.MapStringStringCmd, 0, len(keys))
	for _, key := range keys {
		cmd := tx.HGetAll(ctx, getRecordKey(key))
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
			key.Unwrap()[0],
			key.Unwrap()[1],
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

// PutRecords puts records in Redis.
func (kv *RecordKVStore) PutRecords(ctx context.Context, records []*types.Record) error {
	tx := kv.getTx()

	for _, record := range records {
		tuples := record.Tuples()

		for _, tuple := range tuples {
			flatkey, attr := getAttrKey(tuple.Key())
			encodedValue, err := xdbkv.EncodeValue(tuple.Value())
			if err != nil {
				return err
			}

			tx.HSet(ctx, flatkey, attr, encodedValue)
		}
	}

	_, err := tx.Exec(ctx)

	return err
}

// DeleteRecords deletes records from Redis.
func (kv *RecordKVStore) DeleteRecords(ctx context.Context, keys []*types.Key) error {
	tx := kv.getTx()

	for _, key := range keys {
		flatkey := getRecordKey(key)
		tx.Del(ctx, flatkey)
	}

	_, err := tx.Exec(ctx)

	return err
}

func (kv *RecordKVStore) getTx() redis.Pipeliner {
	return kv.db.Pipeline()
}

func getRecordKey(key *types.Key) string {
	return strings.Join(key.Unwrap(), ":")
}

func getAttrKey(key *types.Key) (string, string) {
	parts := key.Unwrap()

	attr := parts[len(parts)-1]
	flatkey := strings.Join(parts[:len(parts)-1], ":")

	return flatkey, attr
}
