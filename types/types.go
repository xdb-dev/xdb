package types

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

// A single struct to represent all types
type Type struct {
	id          TypeID
	keyTypeID   TypeID
	valueTypeID TypeID
}

func NewType(id TypeID) Type {
	return Type{id: id}
}

func NewArrayType(valueTypeID TypeID) Type {
	return Type{
		id:          TypeIDArray,
		valueTypeID: valueTypeID,
	}
}

func NewMapType(keyTypeID, valueTypeID TypeID) Type {
	return Type{
		id:          TypeIDMap,
		keyTypeID:   keyTypeID,
		valueTypeID: valueTypeID,
	}
}

func (t Type) ID() TypeID        { return t.id }
func (t Type) Name() string      { return typeNames[t.id] }
func (t Type) KeyType() TypeID   { return t.keyTypeID }
func (t Type) ValueType() TypeID { return t.valueTypeID }

var (
	BooleanType  = NewType(TypeIDBoolean)
	IntegerType  = NewType(TypeIDInteger)
	UnsignedType = NewType(TypeIDUnsigned)
	FloatType    = NewType(TypeIDFloat)
	StringType   = NewType(TypeIDString)
	BytesType    = NewType(TypeIDBytes)
	TimeType     = NewType(TypeIDTime)
)
