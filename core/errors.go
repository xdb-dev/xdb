package core

import (
	"errors"
	"maps"

	xerrors "github.com/gojekfarm/xtools/errors"
)

// Sentinel errors shared across the store and RPC layers.
// Defined in core to avoid circular dependencies between packages.
//
// Every XDB error message opens with [xdb/<pkg>], naming the package that
// detected the fault. These sentinels carry the core spelling of it even
// though the layers that return them sit above core. Match with
// [errors.Is] rather than on the message text.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("[xdb/core] not found")

	// ErrAlreadyExists is returned when creating a resource that already exists.
	ErrAlreadyExists = errors.New("[xdb/core] already exists")

	// ErrSchemaViolation is returned when data violates a schema constraint.
	ErrSchemaViolation = errors.New("[xdb/core] schema violation")

	// ErrAttrNotFound is returned when reading an attribute that has no tuple.
	//
	// It is deliberately standalone and must NOT wrap [ErrNotFound]: an
	// attribute typo bubbling out of a handler must not be mapped to the
	// resource-not-found RPC code.
	ErrAttrNotFound = errors.New("[xdb/core] attribute not found")

	// ErrConflict is returned when an optimistic-concurrency compare-and-swap
	// fails: the expected base revision of the caller does not match the
	// stored revision. It is the shared conflict sentinel for schema revision
	// updates and for record _version preconditions. It is deliberately
	// standalone and must NOT wrap [ErrNotFound].
	ErrConflict = errors.New("[xdb/core] revision conflict")

	// ErrUniqueViolation is returned when a write violates a field's unique
	// constraint: another record already holds the same value for a field
	// declared unique. It is a write-time conflict distinct from the
	// revision [ErrConflict], and is enforced only by backends that
	// materialize a unique index (currently SQLite column tables).
	ErrUniqueViolation = errors.New("[xdb/core] unique constraint violation")

	// ErrInvalidFilter is returned when a filter expression is empty or fails
	// to parse, type-check, or compile.
	ErrInvalidFilter = errors.New("[xdb/core] invalid filter")

	// ErrNotImplemented is returned when a requested operation is not
	// implemented, e.g. by a driver or an interim stub.
	ErrNotImplemented = errors.New("[xdb/core] not implemented")

	// ErrInvalidURI is returned when an invalid URI is encountered.
	ErrInvalidURI = errors.New("[xdb/core] invalid URI")

	// ErrUnknownType is returned when an unknown type is encountered.
	ErrUnknownType = errors.New("[xdb/core] unknown type")

	// ErrUnsupportedValue is returned when a value has no XDB type.
	ErrUnsupportedValue = errors.New("[xdb/core] unsupported value")

	// ErrTypeMismatch is returned when a value is not of the expected type.
	ErrTypeMismatch = errors.New("[xdb/core] type mismatch")
)

// ErrorTags returns the structured tags attached to err with xerrors.Wrap,
// or nil if err carries none. The Error tags section of the package doc
// lists the permitted keys and what each one means.
//
// The result is a copy. xerrors edits the tag map of an existing tagged
// error in place, so handing out the original would let a caller rewrite
// the error itself.
func ErrorTags(err error) map[string]string {
	var tagged *xerrors.ErrorTags
	if !errors.As(err, &tagged) {
		return nil
	}

	tags := tagged.All()
	if len(tags) == 0 {
		return nil
	}

	return maps.Clone(tags)
}
