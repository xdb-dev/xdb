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
)
