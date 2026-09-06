package xdbjson

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Unmarshal encodes a [core.Record] into a JSON document.
//
// Dotted attributes unflatten into nested objects and object arrays render as
// arrays of nested objects. The record ID is always emitted; the namespace and
// schema only with [WithIncludeNS] and [WithIncludeSchema].
//
//	data, err := xdbjson.Unmarshal(record, xdbjson.WithIndent("", "  "))
//	data, err := xdbjson.Unmarshal(record, xdbjson.WithFields("name", "email"))
func Unmarshal(record *core.Record, opts ...Option) ([]byte, error) {
	if record == nil {
		return nil, ErrNilRecord
	}

	o := applyOptions(opts)
	data := buildMap(record, o)

	if len(o.fields) > 0 {
		keep := make(map[string]bool, len(o.fields)+1)
		keep[o.idField] = true
		for _, f := range o.fields {
			keep[f] = true
		}

		for k := range data {
			if !keep[k] {
				delete(data, k)
			}
		}
	}

	if o.indent != "" {
		return json.MarshalIndent(data, o.prefix, o.indent)
	}

	return json.Marshal(data)
}

func buildMap(record *core.Record, o options) map[string]any {
	result := make(map[string]any)

	uri := record.URI()
	result[o.idField] = uri.ID()

	if o.includeNS {
		result[o.nsField] = uri.NS()
	}

	if o.includeSchema {
		result[o.schemaField] = uri.Schema()
	}

	tuples := record.Tuples()
	sort.Slice(tuples, func(i, j int) bool {
		return tuples[i].Attr() < tuples[j].Attr()
	})

	for _, tuple := range tuples {
		attr := tuple.Attr()
		value := convertValue(tuple.Value())
		setNested(result, attr, value)
	}

	return result
}

func setNested(m map[string]any, path string, value any) {
	parts := strings.Split(path, ".")

	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		existing, ok := m[key]
		if !ok {
			child := make(map[string]any)
			m[key] = child
			m = child
			continue
		}
		child, ok := existing.(map[string]any)
		if !ok {
			// Overwrite non-map value with a nested map.
			child = make(map[string]any)
			m[key] = child
		}
		m = child
	}

	m[parts[len(parts)-1]] = value
}

// convertValue converts a [core.Value] to its JSON representation.
// The type switch guarantees each As* call matches the value's type, so
// their (impossible) errors are discarded.
func convertValue(v *core.Value) any {
	if v == nil || v.IsNil() {
		return nil
	}

	switch v.Type().ID() {
	case core.TIDBoolean:
		b, _ := v.AsBool()
		return b
	case core.TIDInteger:
		i, _ := v.AsInt()
		return i
	case core.TIDUnsigned:
		u, _ := v.AsUint()
		return u
	case core.TIDFloat:
		f, _ := v.AsFloat()
		return f
	case core.TIDString:
		s, _ := v.AsStr()
		return s
	case core.TIDBytes:
		b, _ := v.AsBytes()
		return base64.StdEncoding.EncodeToString(b)
	case core.TIDTime:
		ts, _ := v.AsTime()
		return ts.Format(time.RFC3339)
	case core.TIDJSON:
		return convertJSON(v)
	case core.TIDArray:
		return convertArray(v)
	default:
		return nil
	}
}

// convertJSON renders a JSON value as its natural Go representation so that
// object-array elements (and plain JSON fields) encode as nested objects rather
// than opaque strings. Member values are already in typed JSON form
// (time.Time as RFC3339, []byte as base64), so they round-trip unchanged.
func convertJSON(v *core.Value) any {
	raw, _ := v.AsJSON()
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw)
	}
	return out
}

func convertArray(v *core.Value) []any {
	raw, _ := v.AsArray()
	result := make([]any, len(raw))
	for i, elem := range raw {
		result[i] = convertValue(elem)
	}
	return result
}
