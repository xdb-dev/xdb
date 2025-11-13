package core

type Builder struct {
	ns     string
	schema string
	id     string
	attr   string
}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) NS(ns string) *Builder {
	b.ns = ns
	return b
}

func (b *Builder) Schema(schema string) *Builder {
	b.schema = schema
	return b
}

func (b *Builder) ID(id string) *Builder {
	b.id = id
	return b
}

func (b *Builder) Attr(attr string) *Builder {
	b.attr = attr
	return b
}

func (b *Builder) URI() (*URI, error) {
	ns, err := ParseNS(b.ns)
	if err != nil {
		return nil, err
	}
	schema, err := ParseSchema(b.schema)
	if err != nil {
		return nil, err
	}
	id, err := ParseID(b.id)
	if err != nil {
		return nil, err
	}
	attr, err := ParseAttr(b.attr)
	if err != nil {
		return nil, err
	}
	return &URI{
		ns:     ns,
		schema: schema,
		id:     id,
		attr:   attr,
	}, nil
}

func (b *Builder) MustURI() *URI {
	uri, err := b.URI()
	if err != nil {
		panic(err)
	}
	return uri
}

func (b *Builder) Record() (*Record, error) {
	uri, err := b.URI()
	if err != nil {
		return nil, err
	}
	return newRecord(uri), nil
}

func (b *Builder) MustRecord() *Record {
	record, err := b.Record()
	if err != nil {
		panic(err)
	}
	return record
}

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

func (b *Builder) MustTuple(attr string, value any) *Tuple {
	tuple, err := b.Tuple(attr, value)
	if err != nil {
		panic(err)
	}
	return tuple
}
