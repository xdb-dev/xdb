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

	result[e.opts.idField] = record.ID().String()

	if e.opts.includeNS {
		result[e.opts.nsField] = record.NS().String()
	}

	if e.opts.includeSchema {
		result[e.opts.schemaField] = record.Schema().String()
	}

	tuples := record.Tuples()
	sort.Slice(tuples, func(i, j int) bool {
		return tuples[i].Attr().String() < tuples[j].Attr().String()
	})

	for _, tuple := range tuples {
		attr := tuple.Attr().String()
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

func convertValue(v *core.Value) any {
	if v == nil || v.IsNil() {
		return nil
	}

	switch v.Type().ID() {
	case core.TIDBoolean:
		return v.MustBool()
	case core.TIDInteger:
		return v.MustInt()
	case core.TIDUnsigned:
		return v.MustUint()
	case core.TIDFloat:
		return v.MustFloat()
	case core.TIDString:
		return v.MustStr()
	case core.TIDBytes:
		return base64.StdEncoding.EncodeToString(v.MustBytes())
	case core.TIDTime:
		return v.MustTime().Format(time.RFC3339)
	case core.TIDArray:
		return convertArray(v)
	default:
		return nil
	}
}

func convertArray(v *core.Value) []any {
	raw := v.MustArray()
	result := make([]any, len(raw))
	for i, elem := range raw {
		result[i] = convertValue(elem)
	}
	return result
}
