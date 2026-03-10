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
)

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

		if field.Type != tuple.Value().Type().ID() {
			return errors.Wrap(ErrTypeMismatch,
				"field", attr,
				"expected", field.Type.String(),
				"got", tuple.Value().Type().ID().String(),
			)
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
