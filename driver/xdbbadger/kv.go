// Package xdbbadger provides a BadgerDB-backed driver for XDB.
package xdbbadger

import (
	"context"
	"strings"

	"github.com/dgraph-io/badger/v4"

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

// KVStore is a key-value store for BadgerDB.
// It stores tuples in BadgerDB as key-value pairs using composite keys.
type KVStore struct {
	db    *badger.DB
	codec codec.KeyValueCodec
}

// New creates a new BadgerDB KVStore.
func New(db *badger.DB) *KVStore {
	return &KVStore{db: db, codec: msgpack.New()}
}

// GetTuples gets tuples from the BadgerDB key-value store.
func (kv *KVStore) GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	tuples := make([]*core.Tuple, 0, len(keys))
	missing := make([]*core.Key, 0)

	err := kv.db.View(func(txn *badger.Txn) error {
		for _, key := range keys {
			compositeKey, err := kv.codec.MarshalKey(key)
			if err != nil {
				return err
			}

			item, err := txn.Get(compositeKey)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					missing = append(missing, key)
					continue
				}
				return err
			}

			var valueBytes []byte
			err = item.Value(func(val []byte) error {
				valueBytes = make([]byte, len(val))
				copy(valueBytes, val)
				return nil
			})
			if err != nil {
				return err
			}

			value, err := kv.codec.UnmarshalValue(valueBytes)
			if err != nil {
				return err
			}

			tuple := key.Value(value)
			tuples = append(tuples, tuple)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return tuples, missing, nil
}

// PutTuples puts tuples into the BadgerDB key-value store.
func (kv *KVStore) PutTuples(ctx context.Context, tuples []*core.Tuple) error {
	if len(tuples) == 0 {
		return nil
	}

	return kv.db.Update(func(txn *badger.Txn) error {
		for _, tuple := range tuples {
			compositeKey, err := kv.codec.MarshalKey(tuple.Key())
			if err != nil {
				return err
			}

			valueBytes, err := kv.codec.MarshalValue(tuple.Value())
			if err != nil {
				return err
			}

			err = txn.Set(compositeKey, valueBytes)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteTuples deletes tuples from the BadgerDB key-value store.
func (kv *KVStore) DeleteTuples(ctx context.Context, keys []*core.Key) error {
	if len(keys) == 0 {
		return nil
	}

	return kv.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			compositeKey, err := kv.codec.MarshalKey(key)
			if err != nil {
				return err
			}
			err = txn.Delete(compositeKey)
			if err != nil && err != badger.ErrKeyNotFound {
				return err
			}
		}
		return nil
	})
}

// GetRecords gets records from the BadgerDB key-value store.
func (kv *KVStore) GetRecords(ctx context.Context, keys []*core.Key) ([]*core.Record, []*core.Key, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	recordsMap := make(map[string]*core.Record)
	missing := make([]*core.Key, 0)

	err := kv.db.View(func(txn *badger.Txn) error {
		for _, key := range keys {
			recordKey, err := kv.codec.MarshalKey(key)
			if err != nil {
				return err
			}

			prefix := []byte(string(recordKey) + ":")

			opts := badger.DefaultIteratorOptions
			opts.Prefix = prefix
			it := txn.NewIterator(opts)
			defer it.Close()

			record := core.NewRecord(key.Kind(), key.ID())
			found := false

			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				compositeKeyBytes := item.Key()

				// Parse the composite key to extract the attribute
				compositeKeyStr := string(compositeKeyBytes)
				parts := strings.Split(compositeKeyStr, ":")
				if len(parts) != 3 {
					continue
				}
				attr := parts[2]

				var valueBytes []byte
				err := item.Value(func(val []byte) error {
					valueBytes = make([]byte, len(val))
					copy(valueBytes, val)
					return nil
				})
				if err != nil {
					return err
				}

				value, err := kv.codec.UnmarshalValue(valueBytes)
				if err != nil {
					return err
				}

				record.Set(attr, value)
				found = true
			}

			if !found {
				missing = append(missing, key)
			} else {
				recordsMap[string(recordKey)] = record
			}
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	records := make([]*core.Record, 0, len(keys)-len(missing))
	for _, key := range keys {
		recordKey, err := kv.codec.MarshalKey(key)
		if err != nil {
			return nil, nil, err
		}

		if record, ok := recordsMap[string(recordKey)]; ok {
			records = append(records, record)
		}
	}

	return records, missing, nil
}

// PutRecords puts records into the BadgerDB key-value store.
func (kv *KVStore) PutRecords(ctx context.Context, records []*core.Record) error {
	tuples := make([]*core.Tuple, 0)
	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}
	return kv.PutTuples(ctx, tuples)
}

// DeleteRecords deletes records from the BadgerDB key-value store.
func (kv *KVStore) DeleteRecords(ctx context.Context, keys []*core.Key) error {
	if len(keys) == 0 {
		return nil
	}

	return kv.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			recordKey, err := kv.codec.MarshalKey(key)
			if err != nil {
				return err
			}
			prefix := []byte(string(recordKey) + ":")

			opts := badger.DefaultIteratorOptions
			opts.Prefix = prefix
			it := txn.NewIterator(opts)
			defer it.Close()

			// Collect all keys to delete
			keysToDelete := make([][]byte, 0)
			for it.Rewind(); it.Valid(); it.Next() {
				keyBytes := make([]byte, len(it.Item().Key()))
				copy(keyBytes, it.Item().Key())
				keysToDelete = append(keysToDelete, keyBytes)
			}

			// Delete all collected keys
			for _, keyToDelete := range keysToDelete {
				err := txn.Delete(keyToDelete)
				if err != nil && err != badger.ErrKeyNotFound {
					return err
				}
			}
		}
		return nil
	})
}
