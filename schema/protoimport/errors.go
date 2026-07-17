package protoimport

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrNoNamespace is returned when a file descriptor has no proto package and
	// no namespace was supplied with [WithNamespace].
	ErrNoNamespace = errors.New("[xdb/protoimport] no namespace")

	// ErrOneof is returned for a field that participates in a oneof. Unions are a
	// non-goal; the error names the field and its oneof.
	ErrOneof = errors.New("[xdb/protoimport] oneof is not supported")

	// ErrRecursive is returned when a message type refers back to itself through
	// nested or repeated message fields. The error names the cycle; the escape
	// hatch is [WithAllowJSON].
	ErrRecursive = errors.New("[xdb/protoimport] recursive message")

	// ErrRename is returned by [CheckRename] when a re-import carries a field
	// whose proto number matches an existing field under a different name.
	ErrRename = errors.New("[xdb/protoimport] field renamed")

	// ErrUnsupported is returned for a proto construct with no XDB mapping.
	ErrUnsupported = errors.New("[xdb/protoimport] unsupported field")
)
