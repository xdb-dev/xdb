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
	edges  map[string]*Edge
}

// NewRecord creates a new Record.
func NewRecord(kind string, id string) *Record {
	return &Record{kind: kind, id: id}
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
func (r *Record) Set(attr string, value *Value) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tuples[attr] = newTuple(r.kind, r.id, attr, value)
	return r
}

// Get returns the tuple for the given attribute.
func (r *Record) Get(attr string) *Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tuples[attr]
}

// AddEdge adds an edge to the Record.
func (r *Record) AddEdge(name string, target *Key) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edges[name] = NewEdge(r.Key(), name, target)
	return r
}
