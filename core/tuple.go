package core

import (
	"fmt"
	"strings"

	"slices"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrInvalidID is returned when an invalid ID is encountered.
	ErrInvalidID = errors.New("[xdb/core] invalid ID")

	// ErrInvalidAttr is returned when an invalid Attr is encountered.
	ErrInvalidAttr = errors.New("[xdb/core] invalid Attr")
)

// ID represents a hierarchical identifier as a slice of strings.
// For example: ["123"] or ["organization", "abc", "department", "engineering"].
type ID []string

// NewID creates a new ID from the provided string components.
// Each component becomes a level in the hierarchical identifier.
func NewID(raw ...string) ID {
	return ID(raw)
}

// String returns the ID as a forward-slash separated string.
// For example: ["123", "456"] becomes "123/456".
func (i ID) String() string {
	return strings.Join(i, "/")
}

// Equals returns true if this ID is equal to the other ID.
// Comparison is done component-wise.
func (i ID) Equals(other ID) bool {
	return slices.Equal(i, other)
}

// Attr represents an attribute name as an array of strings,
// which supports nested attributes.
// For example: ["name"] for simple attributes or ["profile", "email"] for nested ones.
type Attr []string

// NewAttr creates a new Attr from the provided string components.
// Each component represents a level in the attribute hierarchy.
func NewAttr(raw ...string) Attr {
	return Attr(raw)
}

// String returns the Attr as a dot-separated string.
// For example: ["profile", "email"] becomes "profile.email".
func (a Attr) String() string {
	return strings.Join(a, ".")
}

// Tuple is the core data structure of XDB.
//
// Tuple is an immutable data structure containing:
// - Repo: The repository name.
// - ID: The ID of the tuple.
// - Attr: Name of the attribute.
// - Value: Value of the attribute.
type Tuple struct {
	repo  string
	id    ID
	attr  Attr
	value *Value
}

// NewTuple creates a new Tuple.
func NewTuple(repo string, id, attr, value any) *Tuple {
	return &Tuple{
		repo:  repo,
		id:    newID(id),
		attr:  newAttr(attr),
		value: NewValue(value),
	}
}

// URI returns a reference to the tuple.
func (t *Tuple) URI() *URI {
	return NewURI(t.repo, t.id, t.attr)
}

// Repo returns the repository name of the tuple.
func (t *Tuple) Repo() string {
	return t.repo
}

// ID returns the ID of the tuple.
func (t *Tuple) ID() ID {
	return t.id
}

// Attr returns the attribute name.
func (t *Tuple) Attr() Attr {
	return t.attr
}

// Value returns the value of the attribute.
func (t *Tuple) Value() *Value {
	return t.value
}

// IsNil returns true if the tuple value is nil.
func (t *Tuple) IsNil() bool {
	return t.value == nil || t.value.IsNil()
}

// GoString returns Go syntax of the tuple.
func (t *Tuple) GoString() string {
	return fmt.Sprintf("Tuple(%s, %s, %s, %#v)", t.repo, t.id.String(), t.attr.String(), t.value)
}

func newID(id any) ID {
	switch v := id.(type) {
	case ID:
		return v
	case string:
		return NewID(v)
	case []string:
		return ID(v)
	default:
		panic(errors.Wrap(ErrInvalidID, "id", fmt.Sprintf("%v", id)))
	}
}

func newAttr(attr any) Attr {
	switch v := attr.(type) {
	case Attr:
		return v
	case string:
		return Attr{v}
	case []string:
		return Attr(v)
	default:
		panic(errors.Wrap(ErrInvalidAttr, "attr", fmt.Sprintf("%v", attr)))
	}
}
