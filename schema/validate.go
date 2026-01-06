package schema

import (
	"maps"
	"slices"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrUnknownField is returned when a tuple has an attribute not defined in the schema.
	ErrUnknownField = errors.New("[xdb/schema] unknown field")

	// ErrTypeMismatch is returned when a tuple value type does not match the field type.
	ErrTypeMismatch = errors.New("[xdb/schema] type mismatch")
)

// InferFields returns new field definitions for attributes not yet in the schema.
func InferFields(def *Def, tuples []*core.Tuple) ([]*FieldDef, error) {
	found := make(map[string]*FieldDef)

	for _, tuple := range tuples {
		attr := tuple.Attr().String()
		if def.GetField(attr) != nil {
			continue
		}
		if _, ok := found[attr]; ok {
			continue
		}
		found[attr] = &FieldDef{
			Name: attr,
			Type: tuple.Value().Type(),
		}
	}

	return slices.Collect(maps.Values(found)), nil
}

// ValidateTuples validates tuples against the schema definition.
func ValidateTuples(def *Def, tuples []*core.Tuple) error {
	if def.Mode == ModeFlexible {
		return nil
	}

	for _, tuple := range tuples {
		attr := tuple.Attr().String()
		field := def.GetField(attr)

		if field == nil {
			return errors.Wrap(ErrUnknownField, "field", attr)
		}

		if !field.Type.Equals(tuple.Value().Type()) {
			return errors.Wrap(ErrTypeMismatch,
				"field", attr,
				"expected", field.Type.String(),
				"got", tuple.Value().Type().String(),
			)
		}
	}
	return nil
}
