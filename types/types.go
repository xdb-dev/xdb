package types

import (
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

var (
	// ErrUnknownType is returned when an unknown type is encountered.
	ErrUnknownType = errors.New("unknown type")
)

// TypeID represents the type of a value.
type TypeID int

const (
	TypeUnknown TypeID = iota
	TypeBoolean
	TypeInteger
	TypeUnsigned
	TypeFloat
	TypeString
	TypeBytes
	TypeTime
	TypeArray
	TypeMap
)

var typeNames = map[TypeID]string{
	TypeUnknown:  "UNKNOWN",
	TypeBoolean:  "BOOLEAN",
	TypeInteger:  "INTEGER",
	TypeUnsigned: "UNSIGNED",
	TypeFloat:    "FLOAT",
	TypeString:   "STRING",
	TypeBytes:    "BYTES",
	TypeTime:     "TIME",
	TypeArray:    "ARRAY",
	TypeMap:      "MAP",
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

	return TypeUnknown, errors.Wrap(ErrUnknownType, "type", name)
}

// Type represents a type of a value.
type Type interface {
	ID() TypeID
	Name() string
}

var (
	typeBoolean  = &PrimitiveType{id: TypeBoolean}
	typeInteger  = &PrimitiveType{id: TypeInteger}
	typeUnsigned = &PrimitiveType{id: TypeUnsigned}
	typeFloat    = &PrimitiveType{id: TypeFloat}
	typeString   = &PrimitiveType{id: TypeString}
	typeBytes    = &PrimitiveType{id: TypeBytes}
	typeTime     = &PrimitiveType{id: TypeTime}
)

// PrimitiveType represents a primitive type.
type PrimitiveType struct {
	id TypeID
}

// ID returns the type ID.
func (t *PrimitiveType) ID() TypeID {
	return t.id
}

// Name returns the name of the type.
func (t *PrimitiveType) Name() string {
	return typeNames[t.id]
}

// ArrayType represents an array type.
type ArrayType struct {
	valueType Type
}

// NewArrayType creates a new array type.
func NewArrayType(valueType Type) *ArrayType {
	return &ArrayType{valueType: valueType}
}

// ValueType returns the array value type.
func (t *ArrayType) ValueType() Type {
	return t.valueType
}

// ID returns the type ID.
func (t *ArrayType) ID() TypeID {
	return TypeArray
}

// Name returns the name of the type.
func (t *ArrayType) Name() string {
	return typeNames[TypeArray]
}

// MapType represents a map type.
type MapType struct {
	keyType   Type
	valueType Type
}

// NewMapType creates a new map type.
func NewMapType(keyType, valueType Type) *MapType {
	return &MapType{keyType: keyType, valueType: valueType}
}

// KeyType returns the key type.
func (t *MapType) KeyType() Type {
	return t.keyType
}

// ValueType returns the value type.
func (t *MapType) ValueType() Type {
	return t.valueType
}

// ID returns the type ID.
func (t *MapType) ID() TypeID {
	return TypeMap
}

// Name returns the name of the type.
func (t *MapType) Name() string {
	return typeNames[TypeMap]
}
