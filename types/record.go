package types

import (
	"fmt"
	"sync"
)

// Record is a collection of tuples and edges that share
// the same kind and id.
type Record struct {
	kind string
	id   string

	mu     sync.RWMutex
	tuples map[string]*Tuple
}

// NewRecord creates a new Record.
func NewRecord(kind string, id string) *Record {
	return &Record{
		kind:   kind,
		id:     id,
		tuples: make(map[string]*Tuple),
	}
}

// Key returns a reference to the Record.
func (r *Record) Key() *Key {
	return NewKey(r.kind, r.id)
}

// Kind returns the kind of the Record.
func (r *Record) Kind() string {
	return r.kind
}

// ID returns the id of the Record.
func (r *Record) ID() string {
	return r.id
}

// String returns the string representation of the Record.
func (r *Record) String() string {
	return fmt.Sprintf("Record(%s, %s)", r.kind, r.id)
}

// Set adds a tuple to the Record.
func (r *Record) Set(attr string, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tuples[attr] = NewTuple(r.kind, r.id, attr, value)
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
