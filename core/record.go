package core

import (
	"fmt"
	"sync"
)

// Record is a group of tuples that share the same path (NS + SCHEMA + ID).
// Records are mutable and thread-safe, similar to database rows.
type Record struct {
	path *URI

	mu     sync.RWMutex
	tuples map[string]*Tuple
}

// NewRecord creates a new Record.
func NewRecord(ns, schema, id string) *Record {
	path := New().NS(ns).Schema(schema).ID(id).MustURI()
	return newRecord(path)
}

func newRecord(path *URI) *Record {
	return &Record{
		path:   path,
		tuples: make(map[string]*Tuple),
	}
}

// NS returns the namespace of the record.
func (r *Record) NS() *NS { return r.path.NS() }

// Schema returns the schema of the record.
func (r *Record) Schema() *Schema { return r.path.Schema() }

// ID returns the ID of the record.
func (r *Record) ID() *ID { return r.path.ID() }

// URI returns a URI that references this Record.
func (r *Record) URI() *URI { return r.path }

// SchemaURI returns the schema URI of the record.
func (r *Record) SchemaURI() *URI {
	return &URI{
		ns:     r.path.NS(),
		schema: r.path.Schema(),
	}
}

// GoString returns Go syntax of the Record.
func (r *Record) GoString() string {
	return fmt.Sprintf("Record(%s)", r.path.String())
}

// Set adds or updates a tuple in the Record with the given attribute and value.
// If a tuple with the same attribute already exists, it will be replaced.
func (r *Record) Set(attr string, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, err := ParseAttr(attr)
	if err != nil {
		panic(err)
	}

	t := newTuple(r.path, a, value)
	r.tuples[t.Attr().String()] = t

	return r
}

// Get retrieves the tuple for the given attribute path.
// Returns nil if no tuple exists for the specified attribute.
func (r *Record) Get(attr string) *Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tuples[attr]
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
