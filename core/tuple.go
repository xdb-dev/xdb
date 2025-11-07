package core

import (
	"fmt"
	"regexp"
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

// IsEmpty returns true if the ID has no components.
func (i ID) IsEmpty() bool {
	return len(i) == 0
}

// IsValid returns true if the ID is valid (non-empty and contains no empty strings).
func (i ID) IsValid() bool {
	if len(i) == 0 {
		return false
	}
	for _, component := range i {
		if component == "" {
			return false
		}
	}
	return true
}

// Attr represents an attribute name as an array of strings,
// which supports nested attributes.
// For example: ["name"] for simple attributes or ["profile", "email"] for nested ones.
type Attr []string

// NewAttr creates a new Attr from the provided string components.
// Each component represents a level in the attribute hierarchy.
// If a single string contains dots (e.g., "profile.email"), it will be split
// into separate levels. To create a single-level attribute with a dot in the name,
// use the Attr type directly.
func NewAttr(raw ...string) Attr {
	if len(raw) == 1 && strings.Contains(raw[0], ".") {
		return Attr(strings.Split(raw[0], "."))
	}
	return Attr(raw)
}

// String returns the Attr as a dot-separated string.
// For example: ["profile", "email"] becomes "profile.email".
func (a Attr) String() string {
	return strings.Join(a, ".")
}

// IsEmpty returns true if the Attr has no components.
func (a Attr) IsEmpty() bool {
	return len(a) == 0
}

// IsValid returns true if the Attr is valid (non-empty and contains no empty strings).
func (a Attr) IsValid() bool {
	if len(a) == 0 {
		return false
	}
	for _, component := range a {
		if component == "" {
			return false
		}
	}
	return true
}

// Equals returns true if this Attr is equal to the other Attr.
// Comparison is done component-wise.
func (a Attr) Equals(other Attr) bool {
	return slices.Equal(a, other)
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

// newID is a helper that converts various ID representations into an ID.
// It accepts ID, string, or []string types.
// Panics with ErrInvalidID if the input type is not supported.
func newID(id any) ID {
	switch v := id.(type) {
	case ID:
		return v
	case string:
		id, err := ParseID(v)
		if err != nil {
			panic(err)
		}
		return id
	case []string:
		return ID(v)
	default:
		panic(errors.Wrap(ErrInvalidID, "id", fmt.Sprintf("%v", id)))
	}
}

var idRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ParseID parses a string into an ID.
func ParseID(raw string) (ID, error) {
	if raw == "" {
		return ID([]string{}), nil
	}
	parts := strings.Split(raw, "/")
	for _, part := range parts {
		if !idRegex.MatchString(part) {
			return nil, errors.Wrap(ErrInvalidID, "id", raw)
		}
	}
	return ID(parts), nil
}

// newAttr is a helper that converts various Attr representations into an Attr.
// It accepts Attr, string (supports dot-separated paths), or []string types.
// Panics with ErrInvalidAttr if the input type is not supported.
func newAttr(attr any) Attr {
	switch v := attr.(type) {
	case Attr:
		return v
	case string:
		attr, err := ParseAttr(v)
		if err != nil {
			panic(err)
		}
		return attr
	case []string:
		return Attr(v)
	default:
		panic(errors.Wrap(ErrInvalidAttr, "attr", fmt.Sprintf("%v", attr)))
	}
}

var attrRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ParseAttr parses a string into an Attr.
func ParseAttr(raw string) (Attr, error) {
	if raw == "" {
		return Attr([]string{}), nil
	}
	parts := strings.Split(raw, ".")
	for _, part := range parts {
		if !attrRegex.MatchString(part) {
			return nil, errors.Wrap(ErrInvalidAttr, "attr", raw)
		}
	}
	return Attr(parts), nil
}
