package types

import (
	"fmt"
	"sync"
)

// Record is a collection of tuples and edges that share
// the same kind and id.
type Record struct {
	key *Key

	mu     sync.RWMutex
	tuples map[string]*Tuple
}

// NewRecord creates a new Record.
func NewRecord(key *Key) *Record {
	return &Record{
		key:    key,
		tuples: make(map[string]*Tuple),
	}
}

// Key returns the key of the Record.
func (r *Record) Key() *Key {
	return r.key
}

// GoString returns Go syntax for the Record.
func (r *Record) GoString() string {
	return fmt.Sprintf("Record(%s)", r.key)
}

// Set adds a tuple to the Record.
func (r *Record) Set(attr string, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tuples[attr] = NewTuple(r.Key().New(attr), value)
	return r
}

// Get returns the value for the given attribute.
func (r *Record) Get(attr string) *Value {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tuples[attr]
	if !ok {
		return nil
	}

	return t.Value()
}

// IsEmpty returns true if the Record has no tuples or edges.
func (r *Record) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tuples) == 0
}

// Tuples returns the tuples of the Record.
func (r *Record) Tuples() []*Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tuples := make([]*Tuple, 0, len(r.tuples))
	for _, tuple := range r.tuples {
		tuples = append(tuples, tuple)
	}
	return tuples
}
