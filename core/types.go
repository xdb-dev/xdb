package core

import (
	"github.com/gojekfarm/xtools/errors"
)

// ErrUnknownType is returned when an unknown type is encountered.
var ErrUnknownType = errors.New("[xdb/core] unknown type")

// TID represents the type of a value.
type TID int

// All supported types.
const (
	TIDUnknown TID = iota
	TIDBoolean
	TIDInteger
	TIDUnsigned
	TIDFloat
	TIDString
	TIDBytes
	TIDTime
	TIDArray
	TIDJSON
)
