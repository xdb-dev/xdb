package xdbjson

import "github.com/gojekfarm/xtools/errors"

// Errors returned by the data path ([Marshal], [MarshalInto], [Unmarshal]).
var (
	// ErrNilRecord is returned when a nil record is passed to [Unmarshal] or
	// [MarshalInto].
	ErrNilRecord = errors.New("[xdb/xdbjson] record cannot be nil")

	// ErrInvalidJSON is returned when the input is not valid JSON. Both the
	// data path and [ImportSchema] return it.
	ErrInvalidJSON = errors.New("[xdb/xdbjson] invalid JSON")

	// ErrMissingID is returned when the ID field is not found in the document.
	ErrMissingID = errors.New("[xdb/xdbjson] missing ID field")

	// ErrEmptyID is returned when the ID field is present but empty.
	ErrEmptyID = errors.New("[xdb/xdbjson] ID cannot be empty")

	// ErrMissingNamespace is returned when the namespace cannot be determined.
	// On the data path that means it was neither in the document nor supplied
	// with [WithNS]. On [ImportSchema] it means [WithNS] was not supplied; the
	// importer never derives a namespace from the document.
	ErrMissingNamespace = errors.New("[xdb/xdbjson] no namespace")

	// ErrMissingSchema is returned when the schema name cannot be determined.
	// On the data path that means it was neither in the document nor supplied
	// with [WithSchema]. On [ImportSchema] it means the name could not be
	// resolved from the document (title, $id) and [WithSchema] was not given.
	ErrMissingSchema = errors.New("[xdb/xdbjson] no schema name")
)

// Errors returned by [ImportSchema] for JSON Schema constructs that XDB does
// not map. Each names the JSON pointer to the offending node.
var (
	// ErrInvalidKey is returned when one or more property names do not parse as
	// a single attribute segment, or contain a '.' (ambiguous with the path
	// separator). The error lists every offending key. There is no escaping.
	ErrInvalidKey = errors.New("[xdb/xdbjson] invalid property name")

	// ErrUnion is returned for anyOf/oneOf, which model unions (a non-goal).
	ErrUnion = errors.New("[xdb/xdbjson] anyOf/oneOf is not supported")

	// ErrConflict is returned when an allOf merge (or a nested-object flatten)
	// produces two definitions for the same field key.
	ErrConflict = errors.New("[xdb/xdbjson] conflicting field")

	// ErrCrossDocument is returned for a $ref that points outside the current
	// document. Only same-document pointers (starting with '#') are supported.
	ErrCrossDocument = errors.New("[xdb/xdbjson] cross-document $ref is not supported")

	// ErrCyclicRef is returned when a $ref chain refers back to itself. The
	// escape hatch is [WithOpaqueJSON], which imports the pointer as an opaque
	// JSON field.
	ErrCyclicRef = errors.New("[xdb/xdbjson] cyclic $ref")

	// ErrUnresolvedRef is returned when a same-document $ref points at a node
	// that does not exist.
	ErrUnresolvedRef = errors.New("[xdb/xdbjson] unresolved $ref")

	// ErrUnsupported is returned for a JSON Schema construct with no XDB
	// mapping.
	ErrUnsupported = errors.New("[xdb/xdbjson] unsupported construct")
)
