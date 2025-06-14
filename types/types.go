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

func NewType(id TypeID) Type {
	switch id {
	case TypeBoolean:
		return BooleanType{}
	case TypeInteger:
		return IntegerType{}
	case TypeUnsigned:
		return UnsignedType{}
	case TypeFloat:
		return FloatType{}
	case TypeString:
		return StringType{}
	case TypeBytes:
		return BytesType{}
	case TypeTime:
		return TimeType{}
	default:
		panic(errors.Wrap(ErrUnknownType, "type", id.String()))
	}
}

// BooleanType represents a boolean type.
type BooleanType struct{}

// ID returns the type ID.
func (t BooleanType) ID() TypeID { return TypeBoolean }

// Name returns the name of the type.
func (t BooleanType) Name() string { return typeNames[TypeBoolean] }

// IntegerType represents an integer type.
type IntegerType struct{}

// ID returns the type ID.
func (t IntegerType) ID() TypeID { return TypeInteger }

// Name returns the name of the type.
func (t IntegerType) Name() string { return typeNames[TypeInteger] }

// UnsignedType represents an unsigned integer type.
type UnsignedType struct{}

// ID returns the type ID.
func (t UnsignedType) ID() TypeID { return TypeUnsigned }

// Name returns the name of the type.
func (t UnsignedType) Name() string { return typeNames[TypeUnsigned] }

// FloatType represents a float type.
type FloatType struct{}

// ID returns the type ID.
func (t FloatType) ID() TypeID { return TypeFloat }

// Name returns the name of the type.
func (t FloatType) Name() string { return typeNames[TypeFloat] }

// StringType represents a string type.
type StringType struct{}

// ID returns the type ID.
func (t StringType) ID() TypeID { return TypeString }

// Name returns the name of the type.
func (t StringType) Name() string { return typeNames[TypeString] }

// BytesType represents a bytes type.
type BytesType struct{}

// ID returns the type ID.
func (t BytesType) ID() TypeID { return TypeBytes }

// Name returns the name of the type.
func (t BytesType) Name() string { return typeNames[TypeBytes] }

// TimeType represents a time type.
type TimeType struct{}

// ID returns the type ID.
func (t TimeType) ID() TypeID { return TypeTime }

// Name returns the name of the type.
func (t TimeType) Name() string { return typeNames[TypeTime] }

// ArrayType represents an array type.
type ArrayType struct {
	valueType Type
}

// NewArrayType creates a new array type.
func NewArrayType(valueType Type) ArrayType {
	return ArrayType{valueType: valueType}
}

// ValueType returns the array value type.
func (t ArrayType) ValueType() Type {
	return t.valueType
}

// ID returns the type ID.
func (t ArrayType) ID() TypeID {
	return TypeArray
}

// Name returns the name of the type.
func (t ArrayType) Name() string {
	return typeNames[TypeArray]
}

// MapType represents a map type.
type MapType struct {
	keyType   Type
	valueType Type
}

// NewMapType creates a new map type.
func NewMapType(keyType, valueType Type) MapType {
	return MapType{keyType: keyType, valueType: valueType}
}

// KeyType returns the key type.
func (t MapType) KeyType() Type {
	return t.keyType
}

// ValueType returns the value type.
func (t MapType) ValueType() Type {
	return t.valueType
}

// ID returns the type ID.
func (t MapType) ID() TypeID {
	return TypeMap
}

// Name returns the name of the type.
func (t MapType) Name() string {
	return typeNames[TypeMap]
}
