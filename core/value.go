package core

import (
	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrUnsupportedValue is returned when a value is not supported.
	ErrUnsupportedValue = errors.New("[xdb/core] unsupported value")
	// ErrTypeMismatch is returned when a value is not of the expected type.
	ErrTypeMismatch = errors.New("[xdb/core] type mismatch")
)

// Value represents an attribute value using a tagged union.
// A zero Value is considered a nil value.
type Value struct {
	typ  Type
	data any
}
