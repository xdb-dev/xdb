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
	TIDMap
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
	TIDMap:      "MAP",
}

// String returns the name of the type.
func (t TID) String() string {
	return typeNames[t]
}

// ParseType parses a type name into a TID.
func ParseType(name string) (TID, error) {
	name = strings.TrimSpace(strings.ToUpper(name))

	for t, n := range typeNames {
		if n == name {
			return t, nil
		}
	}

	return TIDUnknown, errors.Wrap(ErrUnknownType, "type", name)
}

// Type represents a value type in XDB, including scalar, array, and map types.
type Type struct {
	id          TID
	keyTypeID   TID
	valueTypeID TID
}

// newType creates a new scalar Type with the given TID.
// This is used to create the predefined Type constants.
func newType(id TID) Type {
	return Type{id: id}
}

// NewArrayType returns a new array Type with the given value TID.
func NewArrayType(valueTypeID TID) Type {
	return Type{
		id:          TIDArray,
		valueTypeID: valueTypeID,
	}
}

// NewMapType returns a new map Type with the given key and value TIDs.
func NewMapType(keyTypeID, valueTypeID TID) Type {
	return Type{
		id:          TIDMap,
		keyTypeID:   keyTypeID,
		valueTypeID: valueTypeID,
	}
}

// ID returns the TID of the Type.
func (t Type) ID() TID { return t.id }

// String returns the name of the Type.
func (t Type) String() string { return typeNames[t.id] }

// KeyTypeID returns the key TID for map types.
func (t Type) KeyTypeID() TID { return t.keyTypeID }

// ValueTypeID returns the value TID for array and map types.
func (t Type) ValueTypeID() TID { return t.valueTypeID }

// Equals returns true if this Type is equal to the other Type.
func (t Type) Equals(other Type) bool {
	return t.id == other.id &&
		t.keyTypeID == other.keyTypeID &&
		t.valueTypeID == other.valueTypeID
}

var (
	TypeUnknown  = newType(TIDUnknown)
	TypeBool     = newType(TIDBoolean)
	TypeInt      = newType(TIDInteger)
	TypeUnsigned = newType(TIDUnsigned)
	TypeFloat    = newType(TIDFloat)
	TypeString   = newType(TIDString)
	TypeBytes    = newType(TIDBytes)
	TypeTime     = newType(TIDTime)
)

type jsonType struct {
	ID       string `json:"id"`
	KeyType  string `json:"key_type,omitempty"`
	ElemType string `json:"elem_type,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
func (t Type) MarshalJSON() ([]byte, error) {
	jt := jsonType{
		ID: t.id.String(),
	}
	switch t.id {
	case TIDMap:
		jt.KeyType = t.keyTypeID.String()
		jt.ElemType = t.valueTypeID.String()
	case TIDArray:
		jt.ElemType = t.valueTypeID.String()
	}
	return json.Marshal(jt)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
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

	if jt.KeyType != "" {
		keyTID, err := ParseType(jt.KeyType)
		if err != nil {
			return err
		}
		t.keyTypeID = keyTID
	}

	if jt.ElemType != "" {
		elemTID, err := ParseType(jt.ElemType)
		if err != nil {
			return err
		}
		t.valueTypeID = elemTID
	}

	return nil
}
