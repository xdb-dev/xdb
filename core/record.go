package core

import (
	"fmt"
	"sync"
)

// Record is a collection of tuples and edges that share
// the same kind and id.
type Record struct {
	id ID

	mu     sync.RWMutex
	tuples map[string]*Tuple
}

// NewRecord creates a new Record.
func NewRecord(id ...string) *Record {
	return &Record{
		id:     NewID(id...),
		tuples: make(map[string]*Tuple),
	}
}

// Key returns a reference to the Record.
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

// Set adds a tuple to the Record.
func (r *Record) Set(attr any, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	t := NewTuple(r.id, attr, value)
	r.tuples[t.Attr().String()] = t

	return r
}

// Get returns the tuple for the given attribute.
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
