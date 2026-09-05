package jsonschemaimport

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrInvalidJSON is returned when the input is not valid JSON.
	ErrInvalidJSON = errors.New("[xdb/jsonschemaimport] invalid JSON")

	// ErrNoNamespace is returned when no namespace was supplied with
	// [WithNamespace]. The importer never derives a namespace from the
	// document.
	ErrNoNamespace = errors.New("[xdb/jsonschemaimport] no namespace")

	// ErrNoSchemaName is returned when no schema name can be resolved from the
	// document (title, $id) and none was supplied with [WithSchemaName].
	ErrNoSchemaName = errors.New("[xdb/jsonschemaimport] no schema name")

	// ErrInvalidKey is returned when one or more property names do not parse as
	// a single attribute segment, or contain a '.' (ambiguous with the path
	// separator). The error lists every offending key. There is no escaping.
	ErrInvalidKey = errors.New("[xdb/jsonschemaimport] invalid property name")

	// ErrUnion is returned for anyOf/oneOf, which model unions (a non-goal). The
	// error names the JSON pointer to the offending node.
	ErrUnion = errors.New("[xdb/jsonschemaimport] anyOf/oneOf is not supported")

	// ErrConflict is returned when an allOf merge (or a nested-object flatten)
	// produces two definitions for the same field key.
	ErrConflict = errors.New("[xdb/jsonschemaimport] conflicting field")

	// ErrCrossDocument is returned for a $ref that points outside the current
	// document. Only same-document pointers (starting with '#') are supported.
	ErrCrossDocument = errors.New("[xdb/jsonschemaimport] cross-document $ref is not supported")

	// ErrCyclicRef is returned when a $ref chain refers back to itself. The
	// escape hatch is [WithJSON], which imports the pointer as an opaque JSON
	// field.
	ErrCyclicRef = errors.New("[xdb/jsonschemaimport] cyclic $ref")

	// ErrUnresolvedRef is returned when a same-document $ref points at a node
	// that does not exist.
	ErrUnresolvedRef = errors.New("[xdb/jsonschemaimport] unresolved $ref")

	// ErrUnsupported is returned for a JSON Schema construct with no XDB mapping,
	// naming the JSON pointer to the offending node.
	ErrUnsupported = errors.New("[xdb/jsonschemaimport] unsupported construct")
)
