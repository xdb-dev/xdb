package xdbjson

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// Decoder converts JSON to XDB records.
type Decoder struct {
	opts options
}

// NewDecoder creates a [Decoder] with functional options.
//
//	dec := xdbjson.NewDecoder(xdbjson.WithNS("com.example"), xdbjson.WithSchema("posts"))
func NewDecoder(opts ...Option) *Decoder {
	return &Decoder{opts: applyOptions(opts)}
}

// ToRecord converts JSON bytes to a new core.Record.
func (d *Decoder) ToRecord(data []byte) (*core.Record, error) {
	m, err := d.unmarshal(data)
	if err != nil {
		return nil, err
	}

	id, err := d.extractID(m)
	if err != nil {
		return nil, err
	}

	ns := d.extractNS(m)
	if ns == "" {
		return nil, ErrMissingNamespace
	}

	schema := d.extractSchema(m)
	if schema == "" {
		return nil, ErrMissingSchema
	}

	record := core.NewRecord(ns, schema, id)
	if err := d.populateRecord(record, m); err != nil {
		return nil, err
	}

	return record, nil
}

// ToExistingRecord populates an existing record from JSON bytes.
// The record's NS, Schema, and ID are preserved; only attributes are updated.
func (d *Decoder) ToExistingRecord(data []byte, record *core.Record) error {
	if record == nil {
		return ErrNilRecord
	}

	m, err := d.unmarshal(data)
	if err != nil {
		return err
	}

	return d.populateRecord(record, m)
}

// unmarshal decodes JSON into a map. Numbers decode as [json.Number] so
// integer and float types can be preserved (see [inferNumbers] and
// [convertToType]).
func (d *Decoder) unmarshal(data []byte) (map[string]any, error) {
	var m map[string]any

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		return nil, ErrInvalidJSON
	}
	return m, nil
}

func (d *Decoder) extractID(m map[string]any) (string, error) {
	v, ok := m[d.opts.idField]
	if !ok {
		return "", ErrMissingID
	}

	id, ok := v.(string)
	if !ok {
		id = fmt.Sprintf("%v", v)
	}

	if id == "" {
		return "", ErrEmptyID
	}

	return id, nil
}

func (d *Decoder) extractNS(m map[string]any) string {
	if v, ok := m[d.opts.nsField]; ok {
		if ns, ok := v.(string); ok {
			return ns
		}
	}
	return d.opts.ns
}

func (d *Decoder) extractSchema(m map[string]any) string {
	if v, ok := m[d.opts.schemaField]; ok {
		if schema, ok := v.(string); ok {
			return schema
		}
	}
	return d.opts.schema
}

// populateRecord sets record's attributes from the decoded JSON map m.
//
// A declared field (per [WithDef]) that cannot be typed as its declared type
// is a decode-time error naming the field and expected type — see
// [setDeclaredField]. An undeclared attribute that XDB cannot type (e.g. an
// empty or heterogeneous JSON array) is treated as absent, consistent with
// how null is handled.
func (d *Decoder) populateRecord(record *core.Record, m map[string]any) error {
	flat := make(map[string]any)
	flattenWithDef(m, "", d.opts.def, flat)

	for attr, value := range flat {
		if d.isMetadataField(attr) || value == nil {
			continue
		}

		if d.opts.def != nil {
			if field, ok := d.opts.def.Fields[attr]; ok {
				if err := setDeclaredField(record, attr, value, field); err != nil {
					return err
				}
				continue
			}
		}

		value = inferNumbers(value)

		v, err := core.NewSafeValue(value)
		if err != nil || v == nil {
			continue
		}
		record.Set(attr, v)
	}

	return nil
}

// setDeclaredField converts value per field's declared type and sets it on
// record. Returns an error wrapping [core.ErrSchemaViolation], naming attr and
// field's declared type, when value cannot decode as that type.
func setDeclaredField(record *core.Record, attr string, value any, field schema.Field) error {
	if isObjectArrayField(field) {
		if v, ok := objectArrayValue(value, field.Items); ok {
			record.Set(attr, v)
		}
		return nil
	}

	v, ok := convertDeclaredValue(value, field.Type)
	if !ok {
		return fmt.Errorf(
			"%w: field %q: cannot decode as %s",
			core.ErrSchemaViolation, attr, typeName(field.Type),
		)
	}

	record.Set(attr, v)
	return nil
}

// convertDeclaredValue converts value to type t and wraps the result as a
// [*core.Value]. It reports false when value cannot be represented as t at
// all — a genuine decode failure, as opposed to an undeclared attribute XDB
// simply cannot type.
func convertDeclaredValue(value any, t core.Type) (*core.Value, bool) {
	converted := convertToType(value, t)

	// An empty declared array has no elements to infer a type from; build it
	// directly from the declared element type rather than erroring.
	if t.ID() == core.TIDArray {
		if arr, ok := converted.([]any); ok && len(arr) == 0 {
			return core.ArrayVal(t.ElemTypeID()), true
		}
	}

	v, err := core.NewSafeValue(converted)
	if err != nil || v == nil || !typeMatches(v.Type(), t) {
		return nil, false
	}
	return v, true
}

// typeMatches reports whether got is exactly the declared type want,
// including element type for arrays.
func typeMatches(got, want core.Type) bool {
	if got.ID() != want.ID() {
		return false
	}
	if want.ID() == core.TIDArray {
		return got.ElemTypeID() == want.ElemTypeID()
	}
	return true
}

// typeName renders a field type for error messages, e.g. "INTEGER" or
// "ARRAY<INTEGER>".
func typeName(t core.Type) string {
	if t.ID() == core.TIDArray {
		return "ARRAY<" + t.ElemTypeID().String() + ">"
	}
	return t.ID().String()
}

// inferNumbers converts [json.Number] values into concrete int64 or float64,
// recursing into arrays. A number with a fractional or exponent part becomes a
// float; otherwise it becomes an integer.
func inferNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		s := v.String()
		if strings.ContainsAny(s, ".eE") {
			f, _ := v.Float64()
			return f
		}
		if i, err := v.Int64(); err == nil {
			return i
		}
		f, _ := v.Float64()
		return f
	case []any:
		for i := range v {
			v[i] = inferNumbers(v[i])
		}
		return v
	default:
		return value
	}
}

// isObjectArrayField reports whether field is an ARRAY<JSON> carrying an
// element object schema (Items).
func isObjectArrayField(field schema.Field) bool {
	return field.Type.ID() == core.TIDArray &&
		field.Type.ElemTypeID() == core.TIDJSON &&
		len(field.Items) > 0
}

// objectArrayValue builds an ARRAY<JSON> [core.Value] from a decoded JSON array
// of objects, typing each element's members per items so that time.Time and
// bytes are stored in typed JSON form. Returns false when value is not an array
// of JSON objects.
func objectArrayValue(value any, items map[string]schema.Field) (*core.Value, bool) {
	arr, ok := value.([]any)
	if !ok {
		return nil, false
	}

	elems := make([]*core.Value, 0, len(arr))
	for _, e := range arr {
		raw, ok := convertElement(e, items)
		if !ok {
			return nil, false
		}
		elems = append(elems, core.JSONVal(raw))
	}

	return core.ArrayVal(core.TIDJSON, elems...), true
}

// convertElement types the members of a single object-array element per items
// and re-marshals it to a typed JSON object.
func convertElement(elem any, items map[string]schema.Field) (json.RawMessage, bool) {
	obj, ok := elem.(map[string]any)
	if !ok {
		return nil, false
	}

	out := make(map[string]any, len(obj))
	for k, v := range obj {
		field, ok := items[k]
		if !ok {
			out[k] = v
			continue
		}
		out[k] = convertMember(v, field)
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil, false
	}
	return data, true
}

// convertMember types a single element member. Scalars are converted via
// convertToType (time.Time, bytes, integers); nested object arrays recurse one
// level further via the member's own Items.
func convertMember(v any, field schema.Field) any {
	if isObjectArrayField(field) {
		arr, ok := v.([]any)
		if !ok {
			return v
		}
		out := make([]any, len(arr))
		for i, e := range arr {
			if raw, ok := convertElement(e, field.Items); ok {
				out[i] = raw
			} else {
				out[i] = e
			}
		}
		return out
	}
	return convertToType(v, field.Type)
}

// convertToType converts value to the Go representation of t, always
// returning something that either [core.NewSafeValue] can wrap directly or —
// for an object-array element member — can be re-marshaled to JSON as-is.
// It never returns a [*core.Value]: the same conversion is used both to build
// top-level declared values (via [convertDeclaredValue]) and to type members
// nested inside an object-array element (via [convertMember]), and the latter
// embeds the result straight into a map that gets re-marshaled.
//
// A numeric conversion that would lose precision (e.g. 1.5 into INTEGER) is
// left as its natural numeric type rather than silently truncated, so a
// mismatch stays visible instead of quietly wrong.
func convertToType(value any, t core.Type) any {
	switch t.ID() {
	case core.TIDTime:
		if s, ok := value.(string); ok {
			if ts, err := time.Parse(time.RFC3339, s); err == nil {
				return ts
			}
		}
	case core.TIDInteger:
		if n, ok := value.(json.Number); ok {
			return inferNumbers(n)
		}
	case core.TIDUnsigned:
		if n, ok := value.(json.Number); ok {
			if u, err := strconv.ParseUint(n.String(), 10, 64); err == nil {
				return u
			}
			return inferNumbers(n)
		}
	case core.TIDFloat:
		if n, ok := value.(json.Number); ok {
			if f, err := n.Float64(); err == nil {
				return f
			}
		}
	case core.TIDBytes:
		if s, ok := value.(string); ok {
			if b, err := base64.StdEncoding.DecodeString(s); err == nil {
				return b
			}
		}
	case core.TIDJSON:
		return convertJSONValue(value)
	case core.TIDArray:
		return convertArrayElements(value, t.ElemTypeID())
	}
	return value
}

// convertJSONValue marshals a decoded JSON value to a [json.RawMessage] so it
// can be stored as a JSON-typed value. A value already captured verbatim by
// [flattenWithDef] arrives as a [json.RawMessage] and passes through
// unchanged.
func convertJSONValue(value any) any {
	if raw, ok := value.(json.RawMessage); ok {
		return raw
	}
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	return json.RawMessage(data)
}

// convertArrayElements converts each element of a decoded JSON array to the
// declared element type's Go representation, so that every element ends up
// uniformly typed per the declared elem_type — not per whatever type JSON
// happened to decode the first element as. Returns value unchanged if it is
// not a JSON array.
func convertArrayElements(value any, elemTID core.TID) any {
	arr, ok := value.([]any)
	if !ok {
		return value
	}

	elemType := core.NewType(elemTID)
	elems := make([]any, len(arr))
	for i, e := range arr {
		elems[i] = convertToType(e, elemType)
	}
	return elems
}

func (d *Decoder) isMetadataField(attr string) bool {
	return attr == d.opts.idField ||
		attr == d.opts.nsField ||
		attr == d.opts.schemaField
}

// flattenWithDef flattens nested JSON objects into dot-notation attributes.
// A declared JSON-typed field (per def, top-level or dotted) is captured
// verbatim as a [json.RawMessage] instead of being flattened, so its nested
// keys never turn into synthetic dotted attributes. def may be nil, in which
// case every key recurses or leafs exactly as before.
func flattenWithDef(m map[string]any, prefix string, def *schema.Def, result map[string]any) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if def != nil && value != nil {
			if field, ok := def.Fields[fullKey]; ok && field.Type.ID() == core.TIDJSON {
				result[fullKey] = convertJSONValue(value)
				continue
			}
		}

		switch v := value.(type) {
		case map[string]any:
			flattenWithDef(v, fullKey, def, result)
		default:
			result[fullKey] = value
		}
	}
}
