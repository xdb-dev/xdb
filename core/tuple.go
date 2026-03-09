package core

import (
	"fmt"
)

// Tuple is the core data structure of XDB.
//
// Tuple is an immutable data structure containing:
// - Path: Unique identifier for the record.
// - Attr: Name of the attribute.
// - Value: Value of the attribute.
type Tuple struct {
	path  *URI
	attr  *Attr
	value *Value
}

// NewTuple creates a new Tuple.
func NewTuple(path, attr string, value any) *Tuple {
	p, err := ParsePath(path)
	if err != nil {
		panic(err)
	}
	a, err := ParseAttr(attr)
	if err != nil {
		panic(err)
	}
	return newTuple(p, a, value)
}

func newTuple(path *URI, attr *Attr, value any) *Tuple {
	return &Tuple{
		path:  path,
		attr:  attr,
		value: NewValue(value),
	}
}

// NS returns the namespace of the tuple.
func (t *Tuple) NS() *NS {
	return t.path.NS()
}

// Schema returns the schema of the tuple.
func (t *Tuple) Schema() *Schema {
	return t.path.Schema()
}

// SchemaURI returns the schema URI of the tuple.
func (t *Tuple) SchemaURI() *URI {
	return &URI{
		ns:     t.path.NS(),
		schema: t.path.Schema(),
	}
}

// ID returns the record ID of the tuple.
func (t *Tuple) ID() *ID {
	return t.path.ID()
}

// Path returns the URI that references the record.
func (t *Tuple) Path() *URI {
	return t.path
}

// Attr returns the attribute of the tuple.
func (t *Tuple) Attr() *Attr {
	return t.attr
}

// Value returns the value of the tuple.
func (t *Tuple) Value() *Value {
	return t.value
}

// URI returns a URI that references the tuple.
func (t *Tuple) URI() *URI {
	return &URI{
		ns:     t.path.NS(),
		schema: t.path.Schema(),
		id:     t.path.ID(),
		attr:   t.attr,
	}
}

// GoString returns Go syntax of the tuple.
func (t *Tuple) GoString() string {
	return fmt.Sprintf("Tuple(%s, %s, %#v)", t.path.String(), t.attr.String(), t.value)
}
