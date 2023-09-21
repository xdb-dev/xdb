package xdb

// Tuple is a tuple of a key, name and value.
type Tuple struct {
	typ   Type
	key   *Key
	name  string
	value *Value
}

// NewAttr creates a new attribute.
func NewAttr(key *Key, name string, value *Value) *Tuple {
	return &Tuple{
		typ:   TypeAttr,
		key:   key,
		name:  name,
		value: value,
	}
}

// NewEdge creates a new edge.
func NewEdge(k *Key, name string, target *Key) *Tuple {
	return &Tuple{
		typ:   TypeEdge,
		key:   k,
		name:  name,
		value: key(target),
	}
}

// Type returns the tuple's type.
func (a *Tuple) Type() Type {
	return a.typ
}

// Key returns the tuple's key.
func (a *Tuple) Key() *Key {
	return a.key
}

// Name returns the tuple's name.
func (a *Tuple) Name() string {
	return a.name
}

// Value returns the tuple's value.
func (a *Tuple) Value() *Value {
	return a.value
}

// Ref returns a reference to the tuple.
func (a *Tuple) Ref() *Ref {
	return AttrRef(a.key, a.name)
}
