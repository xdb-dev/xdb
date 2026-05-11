package schema

import (
	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrUnknownField is returned when a tuple has an attribute not defined in the schema.
	ErrUnknownField = errors.New("[xdb/schema] unknown field")

	// ErrTypeMismatch is returned when a tuple value type does not match the field type.
	ErrTypeMismatch = errors.New("[xdb/schema] type mismatch")

	// ErrInvalidField is returned when a field declaration is malformed,
	// e.g. an array field missing its element type.
	ErrInvalidField = errors.New("[xdb/schema] invalid field")

	// ErrImmutableField is returned when a schema update attempts to change
	// an immutable property of an existing field (Type or array ElemType).
	ErrImmutableField = errors.New("[xdb/schema] immutable field")
)

// Validate checks that the schema definition is well-formed. Every array
// field must declare an element type, regardless of mode.
func (d *Def) Validate() error {
	for name, field := range d.Fields {
		if field.Type == core.TIDArray && field.ElemType == "" {
			return errors.Wrap(ErrInvalidField,
				"field", name,
				"reason", "array field requires elem_type",
			)
		}
	}
	return nil
}

// ValidateUpdate checks that updated is a compatible evolution of existing.
// Fields that appear in both must not change Type, and array fields must not
// change ElemType once set. Adding new fields and removing existing fields
// are allowed.
func ValidateUpdate(existing, updated *Def) error {
	for name, oldField := range existing.Fields {
		newField, ok := updated.Fields[name]
		if !ok {
			continue
		}
		if oldField.Type != newField.Type {
			return errors.Wrap(ErrImmutableField,
				"field", name,
				"reason", "type cannot change",
				"from", oldField.Type.String(),
				"to", newField.Type.String(),
			)
		}
		if oldField.Type == core.TIDArray &&
			oldField.ElemType != newField.ElemType {
			return errors.Wrap(ErrImmutableField,
				"field", name,
				"reason", "elem_type cannot change",
				"from", oldField.ElemType.String(),
				"to", newField.ElemType.String(),
			)
		}
	}
	return nil
}

// ValidateTuples validates tuples against the schema definition.
// In [ModeFlexible], returns nil (no validation).
// In [ModeStrict] and [ModeDynamic], checks each tuple's attribute exists
// in the schema and that the value type matches the field type.
func ValidateTuples(def *Def, tuples []*core.Tuple) error {
	if def.Mode == ModeFlexible {
		return nil
	}

	for _, tuple := range tuples {
		attr := tuple.Attr().String()
		field, ok := def.Fields[attr]

		if !ok {
			return errors.Wrap(ErrUnknownField, "field", attr)
		}

		if err := ValidateField(attr, field, tuple.Value()); err != nil {
			return err
		}
	}

	return nil
}

// ValidateRecords validates all tuples in the given records against the schema definition.
func ValidateRecords(def *Def, records []*core.Record) error {
	var tuples []*core.Tuple
	for _, record := range records {
		tuples = append(tuples, record.Tuples()...)
	}
	return ValidateTuples(def, tuples)
}

// ValidateField checks that a value matches a field's declared type, including
// the element type for arrays when the schema specifies one. Returns
// [ErrTypeMismatch] on mismatch.
func ValidateField(attr string, field FieldDef, v *core.Value) error {
	valType := v.Type()
	if field.Type != valType.ID() {
		return errors.Wrap(ErrTypeMismatch,
			"field", attr,
			"expected", field.Type.String(),
			"got", valType.ID().String(),
		)
	}

	if field.Type == core.TIDArray && field.ElemType != valType.ElemTypeID() {
		return errors.Wrap(ErrTypeMismatch,
			"field", attr,
			"expected", "ARRAY<"+field.ElemType.String()+">",
			"got", "ARRAY<"+valType.ElemTypeID().String()+">",
		)
	}

	return nil
}

// InferField returns a [FieldDef] that captures a value's type, including the
// element type for arrays. Used by dynamic-mode schemas to add new fields.
func InferField(v *core.Value) FieldDef {
	t := v.Type()
	f := FieldDef{Type: t.ID()}
	if t.ID() == core.TIDArray {
		f.ElemType = t.ElemTypeID()
	}
	return f
}

// EvolveDynamic validates tuples against a dynamic-mode schema and returns
// any new fields that should be added. Known fields are validated and
// type mismatches return [ErrTypeMismatch]. The returned map is nil if no
// new fields were inferred.
//
// Callers are responsible for applying the returned fields (e.g. DDL,
// persisting the updated schema) in whatever order they require.
func EvolveDynamic(def *Def, tuples []*core.Tuple) (map[string]FieldDef, error) {
	var newFields map[string]FieldDef

	for _, tuple := range tuples {
		attr := tuple.Attr().String()
		field, known := def.Fields[attr]

		if !known {
			if _, dup := newFields[attr]; dup {
				continue
			}
			if newFields == nil {
				newFields = make(map[string]FieldDef)
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
