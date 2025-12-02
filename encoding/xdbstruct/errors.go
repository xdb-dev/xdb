package xdbstruct

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrInvalidInput is returned when the input value is invalid.
	ErrInvalidInput = errors.New("input is invalid")

	// ErrNilPointer is returned when the input is a nil pointer.
	ErrNilPointer = errors.New("input is nil pointer")

	// ErrNotStruct is returned when the input is not a struct or pointer to struct.
	ErrNotStruct = errors.New("input must be a struct or pointer to struct")

	// ErrEmptyNamespace is returned when the namespace is empty.
	ErrEmptyNamespace = errors.New("namespace cannot be empty")

	// ErrEmptySchema is returned when the schema is empty.
	ErrEmptySchema = errors.New("schema cannot be empty")

	// ErrEmptyID is returned when the ID is empty or the primary key field is nil.
	ErrEmptyID = errors.New("id cannot be empty")

	// ErrPrimaryKeyNotString is returned when a primary key is not a string type.
	ErrPrimaryKeyNotString = errors.New("primary key must be a string")

	// ErrNoPrimaryKey is returned when no primary key is found in the struct.
	ErrNoPrimaryKey = errors.New("no primary key found: struct must have a field with primary_key tag or implement IDGetter interface")

	// ErrNoMarshaler is returned when a value does not implement a required marshaler interface.
	ErrNoMarshaler = errors.New("value does not implement marshaler interface")

	// ErrNotPointer is returned when the input is not a pointer.
	ErrNotPointer = errors.New("input must be a pointer to struct")

	// ErrTypeMismatch is returned when a value cannot be converted to the target field type.
	ErrTypeMismatch = errors.New("type mismatch between record value and struct field")

	// ErrNotImplemented is returned when a feature is not yet implemented.
	ErrNotImplemented = errors.New("not implemented")
)
