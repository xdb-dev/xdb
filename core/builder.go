package core

import (
	"encoding/json"
	"time"
)

// Builder provides a fluent API for constructing URIs, Records, and Tuples.
// Use New() to create a new Builder, then chain methods to set components.
type Builder struct {
	ns     string
	schema string
	id     string
	attr   string
}

// New creates a new URI Builder.
// The builder starts empty and each component must be set
// via chaining methods.
func New() *Builder {
	return &Builder{}
}

// NS sets the namespace component for the URI.
func (b *Builder) NS(ns string) *Builder {
	b.ns = ns
	return b
}

// Schema sets the schema component for the URI.
func (b *Builder) Schema(schema string) *Builder {
	b.schema = schema
	return b
}

// ID sets the record ID component for the URI.
func (b *Builder) ID(id string) *Builder {
	b.id = id
	return b
}

// Attr sets the attribute component for the URI.
func (b *Builder) Attr(attr string) *Builder {
	b.attr = attr
	return b
}

// URI builds and returns a URI from the configured components.
// Returns an error if any component is invalid.
func (b *Builder) URI() (*URI, error) {
	var ns *NS
	var schema *Schema
	var id *ID
	var attr *Attr
	var err error

	if b.ns != "" {
		ns, err = ParseNS(b.ns)
		if err != nil {
			return nil, err
		}
	}

	if b.schema != "" {
		schema, err = ParseSchema(b.schema)
		if err != nil {
			return nil, err
		}
	}

	if b.id != "" {
		id, err = ParseID(b.id)
		if err != nil {
			return nil, err
		}
	}

	if b.attr != "" {
		attr, err = ParseAttr(b.attr)
		if err != nil {
			return nil, err
		}
	}

	return &URI{
		ns:     ns,
		schema: schema,
		id:     id,
		attr:   attr,
	}, nil
}

// MustURI is like URI but panics if any component is invalid.
func (b *Builder) MustURI() *URI {
	uri, err := b.URI()
	if err != nil {
		panic(err)
	}
	return uri
}

// Record builds and returns a Record from the configured components.
// Returns an error if any component is invalid.
func (b *Builder) Record() (*Record, error) {
	uri, err := b.URI()
	if err != nil {
		return nil, err
	}
	return newRecord(uri), nil
}

// MustRecord is like Record but panics if any component is invalid.
func (b *Builder) MustRecord() *Record {
	record, err := b.Record()
	if err != nil {
		panic(err)
	}
	return record
}

// Tuple builds and returns a Tuple from the configured components and a
// given attribute/value pair.
// Returns an error if any component is invalid.
func (b *Builder) Tuple(attr string, value any) (*Tuple, error) {
	uri, err := b.URI()
	if err != nil {
		return nil, err
	}

	a, err := ParseAttr(attr)
	if err != nil {
		return nil, err
	}

	return newTuple(uri, a, value), nil
}

// MustTuple is like Tuple but panics if any component is invalid.
func (b *Builder) MustTuple(attr string, value any) *Tuple {
	tuple, err := b.Tuple(attr, value)
	if err != nil {
		panic(err)
	}
	return tuple
}

// typedTuple builds a [Tuple] with a pre-constructed [*Value],
// bypassing reflect-based type inference.
func (b *Builder) typedTuple(attr string, v *Value) (*Tuple, error) {
	uri, err := b.URI()
	if err != nil {
		return nil, err
	}

	a, err := ParseAttr(attr)
	if err != nil {
		return nil, err
	}

	return &Tuple{path: uri, attr: a, value: v}, nil
}

func (b *Builder) mustTypedTuple(attr string, v *Value) *Tuple {
	t, err := b.typedTuple(attr, v)
	if err != nil {
		panic(err)
	}
	return t
}

// Bool builds a boolean [Tuple] with the given attribute name and value.
func (b *Builder) Bool(attr string, v bool) (*Tuple, error) {
	return b.typedTuple(attr, BoolVal(v))
}

// MustBool is like [Builder.Bool] but panics if any component is invalid.
func (b *Builder) MustBool(attr string, v bool) *Tuple {
	return b.mustTypedTuple(attr, BoolVal(v))
}

// Int builds an integer [Tuple] with the given attribute name and value.
func (b *Builder) Int(attr string, v int64) (*Tuple, error) {
	return b.typedTuple(attr, IntVal(v))
}

// MustInt is like [Builder.Int] but panics if any component is invalid.
func (b *Builder) MustInt(attr string, v int64) *Tuple {
	return b.mustTypedTuple(attr, IntVal(v))
}

// Uint builds an unsigned integer [Tuple] with the given attribute name and value.
func (b *Builder) Uint(attr string, v uint64) (*Tuple, error) {
	return b.typedTuple(attr, UintVal(v))
}

// MustUint is like [Builder.Uint] but panics if any component is invalid.
func (b *Builder) MustUint(attr string, v uint64) *Tuple {
	return b.mustTypedTuple(attr, UintVal(v))
}

// Float builds a floating-point [Tuple] with the given attribute name and value.
func (b *Builder) Float(attr string, v float64) (*Tuple, error) {
	return b.typedTuple(attr, FloatVal(v))
}

// MustFloat is like [Builder.Float] but panics if any component is invalid.
func (b *Builder) MustFloat(attr string, v float64) *Tuple {
	return b.mustTypedTuple(attr, FloatVal(v))
}

// Str builds a string [Tuple] with the given attribute name and value.
func (b *Builder) Str(attr, v string) (*Tuple, error) {
	return b.typedTuple(attr, StringVal(v))
}

// MustStr is like [Builder.Str] but panics if any component is invalid.
func (b *Builder) MustStr(attr, v string) *Tuple {
	return b.mustTypedTuple(attr, StringVal(v))
}

// Bytes builds a byte slice [Tuple] with the given attribute name and value.
func (b *Builder) Bytes(attr string, v []byte) (*Tuple, error) {
	return b.typedTuple(attr, BytesVal(v))
}

// MustBytes is like [Builder.Bytes] but panics if any component is invalid.
func (b *Builder) MustBytes(attr string, v []byte) *Tuple {
	return b.mustTypedTuple(attr, BytesVal(v))
}

// Time builds a timestamp [Tuple] with the given attribute name and value.
func (b *Builder) Time(attr string, v time.Time) (*Tuple, error) {
	return b.typedTuple(attr, TimeVal(v))
}

// MustTime is like [Builder.Time] but panics if any component is invalid.
func (b *Builder) MustTime(attr string, v time.Time) *Tuple {
	return b.mustTypedTuple(attr, TimeVal(v))
}

// JSON builds a JSON [Tuple] with the given attribute name and value.
func (b *Builder) JSON(attr string, v json.RawMessage) (*Tuple, error) {
	return b.typedTuple(attr, JSONVal(v))
}

// MustJSON is like [Builder.JSON] but panics if any component is invalid.
func (b *Builder) MustJSON(attr string, v json.RawMessage) *Tuple {
	return b.mustTypedTuple(attr, JSONVal(v))
}
