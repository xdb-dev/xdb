package core

import (
	"fmt"
	"sync"
)

// Record is a collection of tuples that share the same ID.
// Records are mutable and thread-safe, similar to database rows.
type Record struct {
	id ID

	mu     sync.RWMutex
	tuples map[string]*Tuple
}

// NewRecord creates a new Record with the specified ID components.
// The ID is constructed by joining all provided string arguments.
func NewRecord(id ...string) *Record {
	return &Record{
		id:     NewID(id...),
		tuples: make(map[string]*Tuple),
	}
}

// Key returns a Key that references this Record (ID only, no attribute).
func (r *Record) Key() *Key {
	return NewKey(r.id)
}

// ID returns the id of the Record.
func (r *Record) ID() ID {
	return r.id
}

// GoString returns Go syntax of the Record.
func (r *Record) GoString() string {
	return fmt.Sprintf("Record(%s)", r.id.String())
}

// Set adds or updates a tuple in the Record with the given attribute and value.
// If a tuple with the same attribute already exists, it will be replaced.
// Returns the Record to allow method chaining.
func (r *Record) Set(attr any, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := NewTuple(r.id, attr, value)
	r.tuples[t.Attr().String()] = t

	return r
}

// Get retrieves the tuple for the given attribute path.
// Returns nil if no tuple exists for the specified attribute.
func (r *Record) Get(attr ...string) *Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a := NewAttr(attr...)

	return r.tuples[a.String()]
}

// IsEmpty returns true if the Record has no tuples.
func (r *Record) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tuples) == 0
}

// Tuples returns all tuples contained in this Record.
// The returned slice is a copy and safe to modify.
func (r *Record) Tuples() []*Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tuples := make([]*Tuple, 0, len(r.tuples))
	for _, tuple := range r.tuples {
		tuples = append(tuples, tuple)
	}
	return tuples
}
