package badger

import (
	"bytes"
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/vmihailenco/msgpack/v4"
	"github.com/xdb-dev/xdb"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(db *badger.DB) *BadgerStore {
	return &BadgerStore{db: db}
}

func (s *BadgerStore) PutRecord(ctx context.Context, record *xdb.Record) error {
	return s.db.Update(func(txn *badger.Txn) error {
		for _, t := range record.Tuples() {
			k, v, err := encodeTuple(t)
			if err != nil {
				return err
			}

			if err := txn.Set(k, v); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *BadgerStore) GetRecord(ctx context.Context, key *xdb.Key) (*xdb.Record, error) {
	r := xdb.NewRecord(key)

	err := s.db.View(func(txn *badger.Txn) error {
		prefix := recordPrefix(key)
		it := txn.NewIterator(badger.IteratorOptions{
			PrefetchValues: false,
		})
		defer it.Close()

		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			k := item.Key()

			err := item.Value(func(v []byte) error {
				t, err := decodeTuple(key, k, v)
				if err != nil {
					return err
				}

				r.Set(t)

				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return r, err
}

func (s *BadgerStore) DeleteRecord(ctx context.Context, key *xdb.Key) error {
	return s.db.Update(func(txn *badger.Txn) error {
		prefix := recordPrefix(key)
		it := txn.NewIterator(badger.IteratorOptions{
			PrefetchValues: false,
		})
		defer it.Close()

		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			if err := txn.Delete(it.Item().Key()); err != nil {
				return err
			}
		}

		return nil
	})
}

func recordPrefix(key *xdb.Key) string {
	return fmt.Sprintf("%s/%s", key.Kind(), key.ID())
}

func encodeTuple(t *xdb.Tuple) ([]byte, []byte, error) {
	k := fmt.Sprintf("%s/%s/%s", t.Key().Kind(), t.Key().ID(), t.Name())

	v, err := msgpack.Marshal(t.Value())
	if err != nil {
		return nil, nil, err
	}

	return []byte(k), v, nil
}

func decodeTuple(key *xdb.Key, k, v []byte) (*xdb.Tuple, error) {
	parts := bytes.Split(k, []byte("/"))
	if len(parts) != 3 {
		return nil, fmt.Errorf("bad key: %s", k)
	}

	name := string(parts[2])

	var value any
	if err := msgpack.Unmarshal(v, &value); err != nil {
		return nil, err
	}

	// TODO: value type checking
	return xdb.NewTuple(key, name, xdb.Empty{}), nil
}
