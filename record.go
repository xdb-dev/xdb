package xdb

import (
	"sync"

	"golang.org/x/exp/maps"
)

type Record struct {
	key    *Key
	tuples map[string]*Tuple
	mu     sync.RWMutex
}

// NewRecord creates a new record with the given tuples.
func NewRecord(k *Key, tuples ...*Tuple) *Record {
	r := &Record{
		key:    k,
		tuples: make(map[string]*Tuple),
	}

	return r.Set(tuples...)
}

func (r *Record) Key() *Key {
	return r.key
}

func (r *Record) Tuples() []*Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return maps.Values(r.tuples)
}

func (r *Record) Set(tuples ...*Tuple) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, t := range tuples {
		r.tuples[t.Hash()] = t
	}

	return r
}
