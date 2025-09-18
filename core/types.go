package core

import (
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrUnknownType is returned when an unknown type is encountered.
	ErrUnknownType = errors.New("xdb/types: unknown type")
)

// TypeID represents the type of a value.
type TypeID int

// All supported types.
const (
	TypeIDUnknown TypeID = iota
	TypeIDBoolean
	TypeIDInteger
	TypeIDUnsigned
	TypeIDFloat
	TypeIDString
	TypeIDBytes
	TypeIDTime
	TypeIDArray
	TypeIDMap
)

var typeNames = map[TypeID]string{
	TypeIDUnknown:  "UNKNOWN",
	TypeIDBoolean:  "BOOLEAN",
	TypeIDInteger:  "INTEGER",
	TypeIDUnsigned: "UNSIGNED",
	TypeIDFloat:    "FLOAT",
	TypeIDString:   "STRING",
	TypeIDBytes:    "BYTES",
	TypeIDTime:     "TIME",
	TypeIDArray:    "ARRAY",
	TypeIDMap:      "MAP",
}

// String returns the name of the type.
func (t TypeID) String() string {
	return fmt.Sprintf("TypeID(%s)", typeNames[t])
}

// ParseType parses a type name into a TypeID.
func ParseType(name string) (TypeID, error) {
	name = strings.TrimSpace(strings.ToUpper(name))

	for t, n := range typeNames {
		if n == name {
			return t, nil
		}
	}

	return TypeIDUnknown, errors.Wrap(ErrUnknownType, "type", name)
}

// Type represents a value type in XDB, including scalar, array, and map types.
type Type struct {
	id          TypeID
	keyTypeID   TypeID
	valueTypeID TypeID
}

// NewType returns a new Type with the given TypeID.
func NewType(id TypeID) Type {
	return Type{id: id}
}

// NewArrayType returns a new array Type with the given value TypeID.
func NewArrayType(valueTypeID TypeID) Type {
	return Type{
		id:          TypeIDArray,
		valueTypeID: valueTypeID,
	}
}

// NewMapType returns a new map Type with the given key and value TypeIDs.
func NewMapType(keyTypeID, valueTypeID TypeID) Type {
	return Type{
		id:          TypeIDMap,
		keyTypeID:   keyTypeID,
		valueTypeID: valueTypeID,
	}
}

// ID returns the TypeID of the Type.
func (t Type) ID() TypeID { return t.id }

// Name returns the name of the Type.
func (t Type) Name() string { return typeNames[t.id] }

// KeyType returns the key TypeID for map types.
func (t Type) KeyType() TypeID { return t.keyTypeID }

// ValueType returns the value TypeID for array and map types.
func (t Type) ValueType() TypeID { return t.valueTypeID }

var (
	booleanType  = NewType(TypeIDBoolean)
	integerType  = NewType(TypeIDInteger)
	unsignedType = NewType(TypeIDUnsigned)
	floatType    = NewType(TypeIDFloat)
	stringType   = NewType(TypeIDString)
	bytesType    = NewType(TypeIDBytes)
	timeType     = NewType(TypeIDTime)
)
