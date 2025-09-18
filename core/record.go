package core

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
func NewRecord(parts ...string) *Record {
	return &Record{
		key:    NewKey(parts...),
		tuples: make(map[string]*Tuple),
	}
}

// Key returns a reference to the Record.
func (r *Record) Key() *Key {
	return r.key
}

// Kind returns the kind of the Record.
func (r *Record) Kind() string {
	return r.key.Kind()
}

// ID returns the id of the Record.
func (r *Record) ID() string {
	return r.key.ID()
}

// GoString returns Go syntax of the Record.
func (r *Record) GoString() string {
	return fmt.Sprintf("Record(%s)", r.key.String())
}

// Set adds a tuple to the Record.
func (r *Record) Set(attr string, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tuples[attr] = newTuple(r.key.With(attr), value)

	return r
}

// Get returns the tuple for the given attribute.
func (r *Record) Get(attr string) *Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tuples[attr]
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
