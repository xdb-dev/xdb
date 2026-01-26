package xdbjson

import (
	"encoding/json"
	"fmt"

	"github.com/xdb-dev/xdb/core"
)

// Decoder converts JSON to XDB records.
type Decoder struct {
	opts Options
}

// NewDecoder creates a decoder with custom options.
func NewDecoder(opts Options) *Decoder {
	return &Decoder{opts: opts.withDefaults()}
}

// NewDefaultDecoder creates a decoder with default options.
// NS and Schema must be provided as they are required for creating records.
func NewDefaultDecoder(ns, schema string) *Decoder {
	opts := DefaultOptions()
	opts.NS = ns
	opts.Schema = schema
	return NewDecoder(opts)
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
	v, ok := m[d.opts.IDField]
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
	if v, ok := m[d.opts.NSField]; ok {
		if ns, ok := v.(string); ok {
			return ns
		}
	}
	return d.opts.NS
}

func (d *Decoder) extractSchema(m map[string]any) string {
	if v, ok := m[d.opts.SchemaField]; ok {
		if schema, ok := v.(string); ok {
			return schema
		}
	}
	return d.opts.Schema
}

func (d *Decoder) populateRecord(record *core.Record, m map[string]any) {
	flat := make(map[string]any)
	flatten(m, "", flat)

	for attr, value := range flat {
		if d.isMetadataField(attr) {
			continue
		}
		if value != nil {
			record.Set(attr, value)
		}
	}
}

func (d *Decoder) isMetadataField(attr string) bool {
	return attr == d.opts.IDField ||
		attr == d.opts.NSField ||
		attr == d.opts.SchemaField
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
