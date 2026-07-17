package schema

import (
	"strings"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrUnknownField is returned when a tuple has an attribute not defined in the schema.
	ErrUnknownField = errors.New("[xdb/schema] unknown field")

	// ErrTypeMismatch is returned when a tuple value type does not match the field type.
	ErrTypeMismatch = errors.New("[xdb/schema] type mismatch")

	// ErrInvalidField is returned when a field declaration is malformed,
	// e.g. an array field missing its element type, an invalid field name,
	// or a field name that is a path-prefix of another field.
	ErrInvalidField = errors.New("[xdb/schema] invalid field")

	// ErrImmutableField is returned when a schema update attempts to change
	// an immutable property of an existing field (Type or array elem type).
	ErrImmutableField = errors.New("[xdb/schema] immutable field")

	// ErrMissingRequired is returned when a required field has no tuple
	// on a full-record write.
	ErrMissingRequired = errors.New("[xdb/schema] missing required field")
)

// Validate checks that the schema definition is well-formed:
//   - the mode is non-empty and recognized;
//   - every field name parses as an attribute path;
//   - no scalar/JSON field name is a path-prefix of another field;
//   - every array field declares an element type.
func (d *Def) Validate() error {
	if _, ok := validModes[d.Mode]; !ok {
		return errors.Wrap(ErrInvalidMode, "mode", string(d.Mode))
	}

	for name, field := range d.Fields {
		if !validFieldName(name) {
			return errors.Wrap(ErrInvalidField,
				"field", name,
				"reason", "field name is not a valid attribute path",
			)
		}
		if field.Type.ID() == core.TIDArray && field.Type.ElemTypeID() == "" {
			return errors.Wrap(ErrInvalidField,
				"field", name,
				"reason", "array field requires elem_type",
			)
		}
	}

	for outer := range d.Fields {
		prefix := outer + "."
		for inner := range d.Fields {
			if inner == outer {
				continue
			}
			if strings.HasPrefix(inner, prefix) {
				return errors.Wrap(ErrInvalidField,
					"field", outer,
					"conflict", inner,
					"reason", "field name is a path-prefix of another field",
				)
			}
		}
	}

	return nil
}

// validFieldName reports whether name parses as a single attribute path,
// reusing core's attribute validation via [core.ParsePath].
func validFieldName(name string) bool {
	if name == "" {
		return false
	}
	_, err := core.ParsePath("_/_/_#" + name)
	return err == nil
}

// ValidateUpdate checks that updated is a compatible evolution of existing.
// Fields that appear in both must not change Type, and array fields must not
// change their element type once set. Adding new fields and removing existing
// fields are allowed.
func ValidateUpdate(existing, updated *Def) error {
	for name, oldField := range existing.Fields {
		newField, ok := updated.Fields[name]
		if !ok {
			continue
		}
		if oldField.Type.ID() != newField.Type.ID() {
			return errors.Wrap(ErrImmutableField,
				"field", name,
				"reason", "type cannot change",
				"from", oldField.Type.ID().String(),
				"to", newField.Type.ID().String(),
			)
		}
		if oldField.Type.ID() == core.TIDArray &&
			oldField.Type.ElemTypeID() != newField.Type.ElemTypeID() {
			return errors.Wrap(ErrImmutableField,
				"field", name,
				"reason", "elem_type cannot change",
				"from", oldField.Type.ElemTypeID().String(),
				"to", newField.Type.ElemTypeID().String(),
			)
		}
	}
	return nil
}

// ValidateTuples type-checks tuples against the schema's declared fields.
//
// Declared fields are always type-checked. Undeclared attributes are governed
// by the mode: [ModeStrict] rejects them with [ErrUnknownField]; [ModeFlexible]
// ignores them. [ModeDynamic] is handled separately by [EvolveDynamic].
func ValidateTuples(def *Def, tuples []*core.Tuple) error {
	for _, tuple := range tuples {
		attr := tuple.Attr()
		field, ok := def.Fields[attr]

		if !ok {
			if def.Mode == ModeStrict {
				return errors.Wrap(ErrUnknownField, "field", attr)
			}
			continue
		}

		if err := ValidateField(attr, field, tuple.Value()); err != nil {
			return err
		}
	}

	return nil
}

// CheckRequired verifies that every required declared field has a tuple in the
// given slice. An explicit-null tuple satisfies the requirement. Returns
// [ErrMissingRequired] for the first missing field.
//
// Required is a declared-field property and is mode-independent. Callers pass
// the full record's tuples: today all backend write paths are full-record
// replaces, so a Create/Update/Upsert carries every attribute. Partial-patch
// update semantics — where a write may touch a subset of attributes — are
// deferred to the store-API-reshape plan; until then Required must not be
// enforced on a partial tuple slice.
func CheckRequired(def *Def, tuples []*core.Tuple) error {
	present := make(map[string]struct{}, len(tuples))
	for _, tuple := range tuples {
		present[tuple.Attr()] = struct{}{}
	}

	for name, field := range def.Fields {
		if !field.Required {
			continue
		}
		if _, ok := present[name]; !ok {
			return errors.Wrap(ErrMissingRequired, "field", name)
		}
	}

	return nil
}

// ValidateField checks that a value matches a field's declared type, including
// the element type for arrays. Returns [ErrTypeMismatch] on mismatch.
func ValidateField(attr string, field Field, v *core.Value) error {
	want := field.Type
	got := v.Type()

	if want.ID() != got.ID() {
		return errors.Wrap(ErrTypeMismatch,
			"field", attr,
			"expected", want.ID().String(),
			"got", got.ID().String(),
		)
	}

	if want.ID() == core.TIDArray && want.ElemTypeID() != got.ElemTypeID() {
		return errors.Wrap(ErrTypeMismatch,
			"field", attr,
			"expected", "ARRAY<"+want.ElemTypeID().String()+">",
			"got", "ARRAY<"+got.ElemTypeID().String()+">",
		)
	}

	return nil
}

// InferField returns a [Field] that captures a value's type, including the
// element type for arrays. Used by dynamic-mode schemas to add new fields.
func InferField(v *core.Value) Field {
	return Field{Type: v.Type()}
}

// EvolveDynamic validates tuples against a dynamic-mode schema and returns
// any new fields that should be added. Known fields are validated and
// type mismatches return [ErrTypeMismatch]. The returned map is nil if no
// new fields were inferred.
//
// Callers are responsible for applying the returned fields (e.g. DDL,
// persisting the updated schema) in whatever order they require.
func EvolveDynamic(def *Def, tuples []*core.Tuple) (map[string]Field, error) {
	var newFields map[string]Field

	for _, tuple := range tuples {
		attr := tuple.Attr()
		field, known := def.Fields[attr]

		if !known {
			if _, dup := newFields[attr]; dup {
				continue
			}
			if newFields == nil {
				newFields = make(map[string]Field)
			}
			newFields[attr] = InferField(tuple.Value())
			continue
		}

		if err := ValidateField(attr, field, tuple.Value()); err != nil {
			return nil, err
		}
	}

	return newFields, nil
}
