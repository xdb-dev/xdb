package xdbjson

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Encoder converts XDB records to JSON.
type Encoder struct {
	opts options
}

// New creates an [Encoder] with functional options.
//
//	enc := xdbjson.New(xdbjson.WithIncludeNS(), xdbjson.WithIDField("id"))
func New(opts ...Option) *Encoder {
	return &Encoder{opts: applyOptions(opts)}
}

// FromRecord converts a [core.Record] to JSON bytes.
//
// Use [EncodeOption] values to control output format:
//
//	data, err := enc.FromRecord(record, xdbjson.WithIndent("", "  "))
//	data, err := enc.FromRecord(record, xdbjson.WithFields("name", "email"))
func (e *Encoder) FromRecord(record *core.Record, opts ...EncodeOption) ([]byte, error) {
	if record == nil {
		return nil, ErrNilRecord
	}

	var cfg encodeConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	data := e.buildMap(record)

	if len(cfg.fields) > 0 {
		keep := make(map[string]bool, len(cfg.fields)+1)
		keep[e.opts.idField] = true
		for _, f := range cfg.fields {
			keep[f] = true
		}

		for k := range data {
			if !keep[k] {
				delete(data, k)
			}
		}
	}

	if cfg.indent != "" {
		return json.MarshalIndent(data, cfg.prefix, cfg.indent)
	}

	return json.Marshal(data)
}

func (e *Encoder) buildMap(record *core.Record) map[string]any {
	result := make(map[string]any)

	uri := record.URI()
	result[e.opts.idField] = uri.ID()

	if e.opts.includeNS {
		result[e.opts.nsField] = uri.NS()
	}

	if e.opts.includeSchema {
		result[e.opts.schemaField] = uri.Schema()
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
	case core.TIDArray:
		return convertArray(v)
	default:
		return nil
	}
}

func convertArray(v *core.Value) []any {
	raw, _ := v.AsArray()
	result := make([]any, len(raw))
	for i, elem := range raw {
		result[i] = convertValue(elem)
	}
	return result
}
