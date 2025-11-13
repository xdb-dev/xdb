package core

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrInvalidNS is returned when an invalid NS is encountered.
	ErrInvalidNS = errors.New("[xdb/core] invalid NS")
)

// NS identifies the data repository.
type NS struct {
	name string
}

// NewNS creates a new NS from a string.
// Panics if the NS is invalid.
func NewNS(raw string) *NS {
	ns, err := ParseNS(raw)
	if err != nil {
		panic(err)
	}
	return ns
}

// ParseNS parses a string into an NS.
// Returns [ErrInvalidNS] if the NS is invalid.
func ParseNS(raw string) (*NS, error) {
	if !isValidComponent(raw) {
		return nil, ErrInvalidNS
	}
	return &NS{name: raw}, nil
}

// String returns the NS as a string.
func (n *NS) String() string { return n.name }

// Equals returns true if this NS is equal to the other NS.
func (n *NS) Equals(other *NS) bool { return n.name == other.name }
