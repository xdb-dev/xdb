package core

import (
	"encoding/json"
	"strings"

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

var typeNames = map[TID]string{
	TIDUnknown:  "UNKNOWN",
	TIDBoolean:  "BOOLEAN",
	TIDInteger:  "INTEGER",
	TIDUnsigned: "UNSIGNED",
	TIDFloat:    "FLOAT",
	TIDString:   "STRING",
	TIDBytes:    "BYTES",
	TIDTime:     "TIME",
	TIDArray:    "ARRAY",
	TIDJSON:     "JSON",
}

// String returns the name of the type.
func (t TID) String() string {
	return typeNames[t]
}

// ParseType parses a type name into a [TID].
// Returns [ErrUnknownType] if the name is not recognized.
func ParseType(name string) (TID, error) {
	name = strings.TrimSpace(strings.ToUpper(name))

	for t, n := range typeNames {
		if n == name {
			return t, nil
		}
	}

	return TIDUnknown, errors.Wrap(ErrUnknownType, "type", name)
}

// Type represents a value type in XDB, including scalar and array types.
type Type struct {
	id         TID
	elemTypeID TID // only meaningful for TIDArray
}

// newType creates a new scalar [Type] with the given [TID].
func newType(id TID) Type {
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
func (t Type) String() string { return typeNames[t.id] }

// ElemTypeID returns the element [TID] for array types.
// Returns [TIDUnknown] for non-array types.
func (t Type) ElemTypeID() TID { return t.elemTypeID }

// Equals returns true if this Type is equal to the other Type.
func (t Type) Equals(other Type) bool {
	return t.id == other.id && t.elemTypeID == other.elemTypeID
}

// Predefined scalar types.
var (
	TypeUnknown  = newType(TIDUnknown)
	TypeBool     = newType(TIDBoolean)
	TypeInt      = newType(TIDInteger)
	TypeUnsigned = newType(TIDUnsigned)
	TypeFloat    = newType(TIDFloat)
	TypeString   = newType(TIDString)
	TypeBytes    = newType(TIDBytes)
	TypeTime     = newType(TIDTime)
	TypeJSON     = newType(TIDJSON)
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
