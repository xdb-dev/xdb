package core

import (
	"encoding/json"
	"fmt"
	"time"
)

// Tuple associates a record path and attribute name with a typed value:
// xdb://ns/schema/id#attr = value. A [Record] groups tuples sharing a path.
//
// Tuple has no setters, but its value can share mutable data with callers.
// Byte slices, JSON bytes, and array elements are not copied. Callers must
// coordinate access to shared data.
//
// A Tuple contains:
//   - Path: the URI identifying the record (NS + SCHEMA + ID).
//   - Attr: the attribute name.
//   - Value: the typed attribute value.
type Tuple struct {
	path  *URI
	value *Value
	attr  string
}

// NewTuple creates a new Tuple.
//
// It panics if path is not a valid record path, if attr is not a valid
// attribute name, or if value is not a supported type. See [NewValue] for
// the supported set, and validate external input before calling.
func NewTuple(path, attr string, value any) *Tuple {
	p, err := ParsePath(path)
	if err != nil {
		panic(err)
	}
	if err := validateComponent("attr", attr, false); err != nil {
		panic(err)
	}
	return newTuple(p, attr, value)
}

func newTuple(path *URI, attr string, value any) *Tuple {
	return &Tuple{
		path:  path,
		attr:  attr,
		value: MustNewValue(value),
	}
}

// Path returns the URI that references the record.
func (t *Tuple) Path() *URI {
	return t.path
}

// Attr returns the attribute of the tuple.
func (t *Tuple) Attr() string {
	return t.attr
}

// Value returns the value of the tuple.
func (t *Tuple) Value() *Value {
	return t.value
}

// URI returns a URI that references the tuple.
func (t *Tuple) URI() *URI {
	return &URI{
		ns:     t.path.ns,
		schema: t.path.schema,
		id:     t.path.id,
		attr:   t.attr,
	}
}

// GoString returns Go syntax of the tuple.
func (t *Tuple) GoString() string {
	return fmt.Sprintf("Tuple(%s, %s, %#v)", t.path.String(), t.attr, t.value)
}

// --- Typed value accessors ---
//
// These delegate to the underlying [Value] and are nil-safe:
// a nil Tuple returns the zero value and [ErrAttrNotFound].

// AsStr returns the tuple's value as a string.
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not a string.
func (t *Tuple) AsStr() (string, error) {
	if t == nil {
		return "", ErrAttrNotFound
	}
	return t.value.AsStr()
}

// AsInt returns the tuple's value as an int64.
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not an integer.
func (t *Tuple) AsInt() (int64, error) {
	if t == nil {
		return 0, ErrAttrNotFound
	}
	return t.value.AsInt()
}

// AsUint returns the tuple's value as a uint64.
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not an unsigned integer.
func (t *Tuple) AsUint() (uint64, error) {
	if t == nil {
		return 0, ErrAttrNotFound
	}
	return t.value.AsUint()
}

// AsFloat returns the tuple's value as a float64.
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not a float.
func (t *Tuple) AsFloat() (float64, error) {
	if t == nil {
		return 0, ErrAttrNotFound
	}
	return t.value.AsFloat()
}

// AsBool returns the tuple's value as a bool.
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not a boolean.
func (t *Tuple) AsBool() (bool, error) {
	if t == nil {
		return false, ErrAttrNotFound
	}
	return t.value.AsBool()
}

// AsBytes returns the tuple's value as a []byte.
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not bytes.
func (t *Tuple) AsBytes() ([]byte, error) {
	if t == nil {
		return nil, ErrAttrNotFound
	}
	return t.value.AsBytes()
}

// AsTime returns the tuple's value as a [time.Time].
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not a timestamp.
func (t *Tuple) AsTime() (time.Time, error) {
	if t == nil {
		return time.Time{}, ErrAttrNotFound
	}
	return t.value.AsTime()
}

// AsJSON returns the tuple's value as a [json.RawMessage].
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not JSON.
func (t *Tuple) AsJSON() (json.RawMessage, error) {
	if t == nil {
		return nil, ErrAttrNotFound
	}
	return t.value.AsJSON()
}

// AsArray returns the tuple's value as a slice of [*Value].
// Returns [ErrAttrNotFound] if the tuple is nil (attribute absent),
// or [ErrTypeMismatch] if the value is not an array.
func (t *Tuple) AsArray() ([]*Value, error) {
	if t == nil {
		return nil, ErrAttrNotFound
	}
	return t.value.AsArray()
}
