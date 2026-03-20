package core

import (
	"encoding/json"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

// ErrUnknownType is returned when an unknown type is encountered.
var ErrUnknownType = errors.New("[xdb/core] unknown type")

// TID represents the type of a value.
// It is a string to allow defining custom type identifiers
// beyond the built-in set.
type TID string

// Built-in type identifiers.
const (
	TIDUnknown  TID = "UNKNOWN"
	TIDBoolean  TID = "BOOLEAN"
	TIDInteger  TID = "INTEGER"
	TIDUnsigned TID = "UNSIGNED"
	TIDFloat    TID = "FLOAT"
	TIDString   TID = "STRING"
	TIDBytes    TID = "BYTES"
	TIDTime     TID = "TIME"
	TIDArray    TID = "ARRAY"
	TIDJSON     TID = "JSON"
)

// builtinTypes is the set of all built-in type identifiers.
var builtinTypes = map[TID]struct{}{
	TIDUnknown:  {},
	TIDBoolean:  {},
	TIDInteger:  {},
	TIDUnsigned: {},
	TIDFloat:    {},
	TIDString:   {},
	TIDBytes:    {},
	TIDTime:     {},
	TIDArray:    {},
	TIDJSON:     {},
}

// ValueTypes is the ordered list of user-facing value type identifiers.
// UNKNOWN is excluded because it is an internal sentinel.
var ValueTypes = []TID{
	TIDString,
	TIDInteger,
	TIDUnsigned,
	TIDFloat,
	TIDBoolean,
	TIDTime,
	TIDBytes,
	TIDJSON,
	TIDArray,
}

// String returns the name of the type.
func (t TID) String() string {
	return string(t)
}

// Lower returns the lowercase name of the type for user-facing output.
func (t TID) Lower() string {
	return strings.ToLower(string(t))
}

// ParseType parses a type name into a [TID].
// Returns [ErrUnknownType] if the name is not a recognized built-in type.
func ParseType(name string) (TID, error) {
	tid := TID(strings.TrimSpace(strings.ToUpper(name)))

	if _, ok := builtinTypes[tid]; !ok {
		return TIDUnknown, errors.Wrap(ErrUnknownType, "type", string(tid))
	}

	return tid, nil
}

// Type represents a value type in XDB, including scalar and array types.
type Type struct {
	id         TID
	elemTypeID TID // only meaningful for TIDArray
}

// NewType creates a new scalar [Type] with the given [TID].
func NewType(id TID) Type {
	return Type{id: id}
}

// NewArrayType returns a new array [Type] with the given element [TID].
func NewArrayType(elemTypeID TID) Type {
	return Type{
		id:         TIDArray,
		elemTypeID: elemTypeID,
	}
}

// ID returns the [TID] of the Type.
func (t Type) ID() TID { return t.id }

// String returns the name of the Type.
func (t Type) String() string { return t.id.String() }

// ElemTypeID returns the element [TID] for array types.
// Returns an empty [TID] for non-array types.
func (t Type) ElemTypeID() TID { return t.elemTypeID }

// Equals returns true if this Type is equal to the other Type.
func (t Type) Equals(other Type) bool {
	return t.id == other.id && t.elemTypeID == other.elemTypeID
}

// Predefined scalar types.
var (
	TypeUnknown  = NewType(TIDUnknown)
	TypeBool     = NewType(TIDBoolean)
	TypeInt      = NewType(TIDInteger)
	TypeUnsigned = NewType(TIDUnsigned)
	TypeFloat    = NewType(TIDFloat)
	TypeString   = NewType(TIDString)
	TypeBytes    = NewType(TIDBytes)
	TypeTime     = NewType(TIDTime)
	TypeJSON     = NewType(TIDJSON)
)

type jsonType struct {
	ID       string `json:"id"`
	ElemType string `json:"elem_type,omitempty"`
}

// MarshalJSON implements the [json.Marshaler] interface.
func (t Type) MarshalJSON() ([]byte, error) {
	jt := jsonType{
		ID: t.id.String(),
	}

	if t.id == TIDArray {
		jt.ElemType = t.elemTypeID.String()
	}

	return json.Marshal(jt)
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (t *Type) UnmarshalJSON(data []byte) error {
	var jt jsonType
	if err := json.Unmarshal(data, &jt); err != nil {
		return err
	}

	tid, err := ParseType(jt.ID)
	if err != nil {
		return err
	}

	t.id = tid

	if jt.ElemType != "" {
		elemTID, err := ParseType(jt.ElemType)
		if err != nil {
			return err
		}
		t.elemTypeID = elemTID
	}

	return nil
}
