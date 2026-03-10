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
	opts Options
}

// NewEncoder creates an encoder with custom options.
func NewEncoder(opts Options) *Encoder {
	return &Encoder{opts: opts.withDefaults()}
}

// NewDefaultEncoder creates an encoder with default options.
// The encoder will NOT include namespace or schema in JSON output.
func NewDefaultEncoder() *Encoder {
	return NewEncoder(DefaultOptions())
}

// FromRecord converts a core.Record to JSON bytes.
func (e *Encoder) FromRecord(record *core.Record) ([]byte, error) {
	if record == nil {
		return nil, ErrNilRecord
	}

	data := e.buildMap(record)
	return json.Marshal(data)
}

// FromRecordIndent converts a core.Record to indented JSON bytes.
func (e *Encoder) FromRecordIndent(record *core.Record, prefix, indent string) ([]byte, error) {
	if record == nil {
		return nil, ErrNilRecord
	}

	data := e.buildMap(record)
	return json.MarshalIndent(data, prefix, indent)
}

func (e *Encoder) buildMap(record *core.Record) map[string]any {
	result := make(map[string]any)

	result[e.opts.IDField] = record.ID().String()

	if e.opts.IncludeNS {
		result[e.opts.NSField] = record.NS().String()
	}

	if e.opts.IncludeSchema {
		result[e.opts.SchemaField] = record.Schema().String()
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
