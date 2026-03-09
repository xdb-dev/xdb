package core

import (
	"encoding/json"

	"github.com/gojekfarm/xtools/errors"
)

// ErrInvalidNS is returned when an invalid NS is encountered.
var ErrInvalidNS = errors.New("[xdb/core] invalid NS")

// NS identifies the namespace.
type NS struct {
	name string
}

// NewNS creates a new namespace identifier.
// Panics if the namespace is invalid (contains characters outside [a-zA-Z0-9._/-]).
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

// MarshalJSON implements the json.Marshaler interface.
func (n *NS) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.name)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (n *NS) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	ns, err := ParseNS(str)
	if err != nil {
		return err
	}
	*n = *ns
	return nil
}
