package xdb

import (
	"fmt"
)

// RefOption allows setting optional parameters for a Ref.
type RefOption interface {
	applyRef(*Ref)
}

func (k *Key) applyRef(r *Ref) {
	r.target = k
}

// Ref is an unique reference to any stored tuple.
// This can be used to refer to:
// - attributes
// - edges
type Ref struct {
	typ    Type
	key    *Key
	name   string
	target *Key
}

// AttrRef creates a new attribute reference.
func AttrRef(key *Key, name string) *Ref {
	return &Ref{
		typ:  TypeAttr,
		key:  key,
		name: name,
	}
}

// EdgeRef creates a new edge reference.
func EdgeRef(key *Key, name string, opts ...RefOption) *Ref {
	r := &Ref{
		typ:  TypeEdge,
		key:  key,
		name: name,
	}

	for _, opt := range opts {
		opt.applyRef(r)
	}

	return r
}

// Type returns the type.
func (r *Ref) Type() Type {
	return r.typ
}

// Key returns the key.
func (r *Ref) Key() *Key {
	return r.key
}

// Name returns the name.
func (r *Ref) Name() string {
	return r.name
}

// Target returns the target.
func (r *Ref) Target() *Key {
	return r.target
}

// String returns a readable representation.
func (r *Ref) String() string {
	switch r.typ {
	case TypeAttr:
		return fmt.Sprintf("Attr %s/%s", r.key, r.name)
	case TypeEdge:
		return fmt.Sprintf("Edge %s -> %s -> %s", r.key, r.name, r.target)
	default:
		return fmt.Sprintf("Unknown Ref %b", r.typ)
	}
}

// Referrer is an interface for storable objects.
type Referrer interface {
	Ref() *Ref
}

// Refs collects and returns the refs of the given Referrers.
func Refs[T Referrer](refs ...T) []*Ref {
	rs := make([]*Ref, len(refs))

	for i, ref := range refs {
		rs[i] = ref.Ref()
	}

	return rs
}

// MakeAttrRefs creates a cross-product of the given keys and names.
func MakeAttrRefs(keys []*Key, names ...string) []*Ref {
	refs := make([]*Ref, len(keys)*len(names))

	for i, key := range keys {
		for j, name := range names {
			refs[i*len(names)+j] = AttrRef(key, name)
		}
	}

	return refs
}
