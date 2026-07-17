package xdbstruct

import "github.com/gojekfarm/xtools/errors"

var (
	// ErrNotStruct is returned when the reflected type is not a struct.
	ErrNotStruct = errors.New("[xdb/xdbstruct] type is not a struct")

	// ErrUnsupported is returned for a field whose type has no XDB mapping and
	// no `json` opt-in. The error names the field and the fix.
	ErrUnsupported = errors.New("[xdb/xdbstruct] unsupported field type")

	// ErrRecursive is returned when a struct type refers back to itself. The
	// error names the field and the cycle; the escape hatch is `xdb:"...,json"`.
	ErrRecursive = errors.New("[xdb/xdbstruct] recursive type")
)

// rejectErr builds an [ErrUnsupported] naming the field, the offending kind,
// and the fix.
func rejectErr(field, kind, fix string) error {
	return errors.Wrap(ErrUnsupported,
		"field", field,
		"kind", kind,
		"fix", fix,
	)
}
