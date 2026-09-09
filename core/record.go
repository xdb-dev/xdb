package core

import (
	"fmt"
	"sync"
)

// Record groups tuples that share a record path (NS + SCHEMA + ID).
// Its methods synchronize access to the tuple map. Values can contain shared
// mutable data; callers must synchronize mutations to that data themselves.
type Record struct {
	path   *URI
	tuples map[string]*Tuple
	mu     sync.RWMutex
}

// NewRecord creates a new Record.
func NewRecord(ns, schema, id string) *Record {
	return newRecord(MustNewURI(ns, schema, id))
}

func newRecord(path *URI) *Record {
	return &Record{
		path:   path,
		tuples: make(map[string]*Tuple),
	}
}

// NewRecordFromTuples builds a Record at path (its attr is dropped)
// from an existing tuple set. It is the read-side inverse of
// [Record.Tuples]: assemble a record view from stored facts.
func NewRecordFromTuples(path *URI, tuples []*Tuple) *Record {
	record := newRecord(path.RecordURI())
	for _, tuple := range tuples {
		record.Set(tuple.Attr(), tuple.Value())
	}
	return record
}

// URI returns a URI that references this Record.
func (r *Record) URI() *URI { return r.path }

// GoString returns Go syntax of the Record.
func (r *Record) GoString() string {
	return fmt.Sprintf("Record(%s)", r.path.String())
}

// Set adds or updates a tuple in the Record with the given attribute and value.
// If a tuple with the same attribute already exists, it will be replaced.
//
// It panics if attr is not a valid attribute name, or if value is not a
// supported type. See [NewValue] for the supported set. Set is the usual
// way to assemble a record from external data, so validate that data with
// [NewValue] first rather than relying on Set to be forgiving.
func (r *Record) Set(attr string, value any) *Record {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validateComponent("attr", attr, false); err != nil {
		panic(err)
	}

	t := newTuple(r.path, attr, value)
	r.tuples[attr] = t

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

// Tuples returns the record's tuples in unspecified order.
// The returned slice can be modified without changing the record's tuple map.
// The tuples and their underlying values remain shared.
func (r *Record) Tuples() []*Tuple {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tuples := make([]*Tuple, 0, len(r.tuples))
	for _, tuple := range r.tuples {
		tuples = append(tuples, tuple)
	}
	return tuples
}
