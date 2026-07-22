package core

import "errors"

// Sentinel errors shared across the store and RPC layers.
// Defined in core to avoid circular dependencies between packages.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("[xdb] not found")

	// ErrAlreadyExists is returned when creating a resource that already exists.
	ErrAlreadyExists = errors.New("[xdb] already exists")

	// ErrSchemaViolation is returned when data violates a schema constraint.
	ErrSchemaViolation = errors.New("[xdb] schema violation")

	// ErrAttrNotFound is returned when reading an attribute that has no tuple.
	//
	// It is deliberately standalone and must NOT wrap [ErrNotFound]: an
	// attribute typo bubbling out of a handler must not be mapped to the
	// resource-not-found RPC code.
	ErrAttrNotFound = errors.New("[xdb/core] attribute not found")

	// ErrConflict is returned when an optimistic-concurrency compare-and-swap
	// fails: the caller's expected base revision does not match the currently
	// stored revision. It is the shared conflict sentinel for schema updates
	// (and, in future, record _rev). It is deliberately standalone and must NOT
	// wrap [ErrNotFound].
	ErrConflict = errors.New("[xdb/core] revision conflict")

	// ErrInvalidFilter is returned when a filter expression is empty or fails
	// to parse, type-check, or compile.
	ErrInvalidFilter = errors.New("[xdb/core] invalid filter")

	// ErrNotImplemented is returned when a requested operation is not
	// implemented, e.g. by a driver or an interim stub.
	ErrNotImplemented = errors.New("[xdb] not implemented")
)
