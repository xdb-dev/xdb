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

	mu       sync.RWMutex
	tuples   map[string]*Tuple
	edges    map[string]*Edge
	edgeKeys map[string][]*Key
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
func (r *Record) Set(attr string, value any) *Record {
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

	if _, ok := r.edgeKeys[name]; !ok {
		r.edgeKeys[name] = make([]*Key, 0)
	}

	r.edgeKeys[name] = append(r.edgeKeys[name], target)

	return r
}

// GetEdge returns the edge's target key.
func (r *Record) GetEdge(name string) *Key {
	r.mu.RLock()
	defer r.mu.RUnlock()

	edgeKeys := r.GetEdges(name)
	if len(edgeKeys) == 0 {
		return nil
	}

	return edgeKeys[0]
}

// GetEdges returns the edges's target keys.
func (r *Record) GetEdges(name string) []*Key {
	r.mu.RLock()
	defer r.mu.RUnlock()

	edgeKeys, ok := r.edgeKeys[name]
	if !ok {
		return nil
	}

	return edgeKeys
}

// IsEmpty returns true if the Record has no tuples or edges.
func (r *Record) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tuples) == 0 && len(r.edges) == 0
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

// Edges returns the edges of the Record.
func (r *Record) Edges() []*Edge {
	r.mu.RLock()
	defer r.mu.RUnlock()

	edges := make([]*Edge, 0, len(r.edges))
	for _, edge := range r.edges {
		edges = append(edges, edge)
	}
	return edges
}
