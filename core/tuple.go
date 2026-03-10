package core

import (
	"encoding/json"
	"fmt"
	"time"
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

// --- Typed value accessors ---
//
// These delegate to the underlying [Value] and are nil-safe:
// a nil Tuple returns the zero value with no error.

// AsStr returns the tuple's value as a string.
// Returns [ErrTypeMismatch] if the value is not a string.
func (t *Tuple) AsStr() (string, error) {
	if t == nil {
		return "", nil
	}
	return t.value.AsStr()
}

// AsInt returns the tuple's value as an int64.
// Returns [ErrTypeMismatch] if the value is not an integer.
func (t *Tuple) AsInt() (int64, error) {
	if t == nil {
		return 0, nil
	}
	return t.value.AsInt()
}

// AsUint returns the tuple's value as a uint64.
// Returns [ErrTypeMismatch] if the value is not an unsigned integer.
func (t *Tuple) AsUint() (uint64, error) {
	if t == nil {
		return 0, nil
	}
	return t.value.AsUint()
}

// AsFloat returns the tuple's value as a float64.
// Returns [ErrTypeMismatch] if the value is not a float.
func (t *Tuple) AsFloat() (float64, error) {
	if t == nil {
		return 0, nil
	}
	return t.value.AsFloat()
}

// AsBool returns the tuple's value as a bool.
// Returns [ErrTypeMismatch] if the value is not a boolean.
func (t *Tuple) AsBool() (bool, error) {
	if t == nil {
		return false, nil
	}
	return t.value.AsBool()
}

// AsBytes returns the tuple's value as a []byte.
// Returns [ErrTypeMismatch] if the value is not bytes.
func (t *Tuple) AsBytes() ([]byte, error) {
	if t == nil {
		return nil, nil
	}
	return t.value.AsBytes()
}

// AsTime returns the tuple's value as a [time.Time].
// Returns [ErrTypeMismatch] if the value is not a timestamp.
func (t *Tuple) AsTime() (time.Time, error) {
	if t == nil {
		return time.Time{}, nil
	}
	return t.value.AsTime()
}

// AsJSON returns the tuple's value as a [json.RawMessage].
// Returns [ErrTypeMismatch] if the value is not JSON.
func (t *Tuple) AsJSON() (json.RawMessage, error) {
	if t == nil {
		return nil, nil
	}
	return t.value.AsJSON()
}

// AsArray returns the tuple's value as a slice of [*Value].
// Returns [ErrTypeMismatch] if the value is not an array.
func (t *Tuple) AsArray() ([]*Value, error) {
	if t == nil {
		return nil, nil
	}
	return t.value.AsArray()
}
