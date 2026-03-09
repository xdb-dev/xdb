package core

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
