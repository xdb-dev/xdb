package schema

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

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
//   - every array field declares an element type;
//   - an ARRAY<JSON> field's Items are validated recursively, and Items is set
//     only on ARRAY<JSON> fields.
func (d *Def) Validate() error {
	if _, ok := validModes[d.Mode]; !ok {
		return errors.Wrap(ErrInvalidMode, "mode", string(d.Mode))
	}

	return validateFields(d.Fields)
}

// validateFields checks a set of declared fields for well-formedness. It is
// used for both the top-level schema fields and, recursively, for the Items of
// object-array fields (which form a separate namespace).
func validateFields(fields map[string]Field) error {
	for name, field := range fields {
		if !validFieldName(name) {
			return errors.Wrap(ErrInvalidField,
				"field", name,
				"reason", "field name is not a valid attribute path",
			)
		}

		isArray := field.Type.ID() == core.TIDArray
		if isArray && field.Type.ElemTypeID() == "" {
			return errors.Wrap(ErrInvalidField,
				"field", name,
				"reason", "array field requires elem_type",
			)
		}

		if len(field.Items) > 0 {
			if !isObjectArray(field) {
				return errors.Wrap(ErrInvalidField,
					"field", name,
					"reason", "items is only valid on ARRAY<JSON> fields",
				)
			}
			if err := validateFields(field.Items); err != nil {
				return err
			}
		}
	}

	for outer := range fields {
		prefix := outer + "."
		for inner := range fields {
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

// isObjectArray reports whether field is an array of JSON objects, i.e. its
// type is ARRAY<JSON>. Only such fields may carry Items.
func isObjectArray(field Field) bool {
	return field.Type.ID() == core.TIDArray &&
		field.Type.ElemTypeID() == core.TIDJSON
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

// NextRevision implements the optimistic-concurrency compare-and-swap for
// schema updates. cur is the currently stored revision; want is the caller's
// expected base revision (typically def.Revision).
//
// A want of 0 means an unconditional update. Otherwise want must equal cur or
// the caller's base is stale and [core.ErrConflict] is returned. On success it
// returns the next revision (cur + 1) to persist.
func NextRevision(cur, want int64) (int64, error) {
	if want != 0 && want != cur {
		return 0, core.ErrConflict
	}
	return cur + 1, nil
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

	if isObjectArray(field) && len(field.Items) > 0 {
		return validateObjectArray(attr, field.Items, v)
	}

	return nil
}

// validateObjectArray checks that every element of an ARRAY<JSON> value is a
// JSON object whose members type-check against items.
func validateObjectArray(attr string, items map[string]Field, v *core.Value) error {
	elems, err := v.AsArray()
	if err != nil {
		return err
	}

	for _, elem := range elems {
		raw, err := elem.AsJSON()
		if err != nil {
			return errors.Wrap(ErrTypeMismatch,
				"field", attr,
				"reason", "array element is not a JSON object",
			)
		}
		if err := validateElement(attr, items, raw); err != nil {
			return err
		}
	}

	return nil
}

// validateElement type-checks a single object-array element against items,
// applying the same rules as top-level fields: unknown members are rejected,
// each present member type-checks, and required members must be present. An
// explicit-null member satisfies the requirement without a type check.
func validateElement(attr string, items map[string]Field, raw json.RawMessage) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return errors.Wrap(ErrTypeMismatch,
			"field", attr,
			"reason", "array element is not a JSON object",
		)
	}

	for name, memberRaw := range obj {
		field, ok := items[name]
		if !ok {
			return errors.Wrap(ErrUnknownField,
				"field", attr,
				"member", name,
			)
		}

		if isJSONNull(memberRaw) {
			continue
		}

		mv, err := jsonMemberValue(memberRaw, field.Type)
		if err != nil {
			return errors.Wrap(err,
				"field", attr,
				"member", name,
			)
		}
		if err := ValidateField(name, field, mv); err != nil {
			return err
		}
	}

	for name, field := range items {
		if !field.Required {
			continue
		}
		if _, ok := obj[name]; !ok {
			return errors.Wrap(ErrMissingRequired,
				"field", attr,
				"member", name,
			)
		}
	}

	return nil
}

// isJSONNull reports whether raw is the JSON null literal.
func isJSONNull(raw json.RawMessage) bool {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return false
	}
	return v == nil
}

// jsonMemberValue converts a JSON object member into a typed [core.Value]
// according to the declared type t. It returns [ErrTypeMismatch] when the JSON
// shape is incompatible with t. Object-array elements (ARRAY<JSON>) are wrapped
// as JSON values so that ValidateField recurses one level further via Items.
func jsonMemberValue(raw json.RawMessage, t core.Type) (*core.Value, error) {
	switch t.ID() {
	case core.TIDJSON:
		return core.JSONVal(raw), nil
	case core.TIDArray:
		var rawElems []json.RawMessage
		if err := json.Unmarshal(raw, &rawElems); err != nil {
			return nil, ErrTypeMismatch
		}
		elemType := core.NewType(t.ElemTypeID())
		elems := make([]*core.Value, 0, len(rawElems))
		for _, re := range rawElems {
			ev, err := jsonMemberValue(re, elemType)
			if err != nil {
				return nil, err
			}
			elems = append(elems, ev)
		}
		return core.ArrayVal(t.ElemTypeID(), elems...), nil
	default:
		return jsonScalarValue(raw, t.ID())
	}
}

// jsonScalarValue converts a JSON scalar into a typed scalar [core.Value].
// String-derived types (STRING, TIME, BYTES) are handled by jsonStringValue.
func jsonScalarValue(raw json.RawMessage, tid core.TID) (*core.Value, error) {
	switch tid {
	case core.TIDBoolean:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, ErrTypeMismatch
		}
		return core.BoolVal(b), nil
	case core.TIDInteger:
		var i int64
		if err := json.Unmarshal(raw, &i); err != nil {
			return nil, ErrTypeMismatch
		}
		return core.IntVal(i), nil
	case core.TIDUnsigned:
		var u uint64
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, ErrTypeMismatch
		}
		return core.UintVal(u), nil
	case core.TIDFloat:
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, ErrTypeMismatch
		}
		return core.FloatVal(f), nil
	default:
		return jsonStringValue(raw, tid)
	}
}

// jsonStringValue converts a JSON string into a STRING, TIME, or BYTES
// [core.Value]. All three require the JSON member to be a string.
func jsonStringValue(raw json.RawMessage, tid core.TID) (*core.Value, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, ErrTypeMismatch
	}

	switch tid {
	case core.TIDString:
		return core.StringVal(s), nil
	case core.TIDTime:
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, ErrTypeMismatch
		}
		return core.TimeVal(ts), nil
	case core.TIDBytes:
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, ErrTypeMismatch
		}
		return core.BytesVal(b), nil
	default:
		return nil, ErrTypeMismatch
	}
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
