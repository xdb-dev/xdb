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
	d.populateRecord(record, m)

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

	d.populateRecord(record, m)
	return nil
}

// unmarshal decodes JSON into a map. With number inference enabled, numbers
// are decoded as [json.Number] so integer and float types can be preserved.
func (d *Decoder) unmarshal(data []byte) (map[string]any, error) {
	var m map[string]any

	if d.opts.numberInference {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(&m); err != nil {
			return nil, ErrInvalidJSON
		}
		return m, nil
	}

	if err := json.Unmarshal(data, &m); err != nil {
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

func (d *Decoder) populateRecord(record *core.Record, m map[string]any) {
	flat := make(map[string]any)
	flatten(m, "", flat)

	for attr, value := range flat {
		if d.isMetadataField(attr) || value == nil {
			continue
		}

		if d.opts.def != nil {
			if field, ok := d.opts.def.Fields[attr]; ok {
				if isObjectArrayField(field) {
					if v, ok := objectArrayValue(value, field.Items); ok {
						record.Set(attr, v)
					}
					continue
				}
				value = convertToType(value, field.Type.ID())
			}
		}

		if d.opts.numberInference {
			value = inferNumbers(value)
		}

		// Skip values XDB cannot type, e.g. an empty or heterogeneous JSON
		// array. Treated as absent, consistent with how null is handled.
		v, err := core.NewSafeValue(value)
		if err != nil || v == nil {
			continue
		}
		record.Set(attr, v)
	}
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
	return convertToType(v, field.Type.ID())
}

func convertToType(value any, fieldType core.TID) any {
	switch fieldType {
	case core.TIDTime:
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
		}
	case core.TIDInteger:
		switch n := value.(type) {
		case float64:
			return int64(n)
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return i
			}
		}
	case core.TIDUnsigned:
		switch n := value.(type) {
		case float64:
			return uint64(n)
		case json.Number:
			if u, err := strconv.ParseUint(n.String(), 10, 64); err == nil {
				return u
			}
		}
	case core.TIDFloat:
		switch n := value.(type) {
		case float64:
			return n
		case json.Number:
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
	}
	return value
}

func (d *Decoder) isMetadataField(attr string) bool {
	return attr == d.opts.idField ||
		attr == d.opts.nsField ||
		attr == d.opts.schemaField
}

func flatten(m map[string]any, prefix string, result map[string]any) {
	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			flatten(v, fullKey, result)
		default:
			result[fullKey] = value
		}
	}
}
