package xdbjson

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xdb-dev/xdb/core"
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
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, ErrInvalidJSON
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

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ErrInvalidJSON
	}

	d.populateRecord(record, m)
	return nil
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
		if d.isMetadataField(attr) {
			continue
		}
		if value != nil {
			if d.opts.def != nil {
				if field, ok := d.opts.def.Fields[attr]; ok {
					value = convertToType(value, field.Type)
				}
			}
			record.Set(attr, value)
		}
	}
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
		if f, ok := value.(float64); ok {
			return int64(f)
		}
	case core.TIDUnsigned:
		if f, ok := value.(float64); ok {
			return uint64(f)
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
