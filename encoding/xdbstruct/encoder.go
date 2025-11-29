package xdbstruct

import (
	"encoding"
	"encoding/json"
	"reflect"

	"github.com/xdb-dev/xdb/core"
)

// Encoder converts Go structs to XDB records.
type Encoder struct {
	opts Options
}

// NewEncoder creates an encoder with custom options.
func NewEncoder(opts Options) *Encoder {
	if opts.Tag == "" {
		opts.Tag = "xdb"
	}
	return &Encoder{opts: opts}
}

// NewDefaultEncoder creates an encoder with default options (Tag: "xdb").
func NewDefaultEncoder(ns, schema string) *Encoder {
	opts := DefaultOptions()
	opts.NS = ns
	opts.Schema = schema
	return NewEncoder(opts)
}

// Encode converts a struct to a core.Record.
// Returns an error if:
//   - v is not a struct or pointer to struct
//   - Cannot determine record ID (no IDGetter interface, no primary_key tag)
//   - ID is empty
//   - Namespace is empty
//   - Schema is empty
//   - Type conversion fails
func (e *Encoder) Encode(v any) (*core.Record, error) {
	rv, err := e.validateInput(v)
	if err != nil {
		return nil, err
	}

	fields, err := e.parseFields(rv, "")
	if err != nil {
		return nil, err
	}

	meta, err := e.resolveMetadata(v, fields)
	if err != nil {
		return nil, err
	}

	return e.buildRecord(meta, fields), nil
}

func (e *Encoder) validateInput(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return rv, ErrInvalidInput
	}

	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return rv, ErrNilPointer
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return rv, ErrNotStruct
	}

	return rv, nil
}

func (e *Encoder) resolveMetadata(v any, fields []field) (metadata, error) {
	m := metadata{ns: e.opts.NS, schema: e.opts.Schema}

	if nsGetter, ok := v.(NSGetter); ok {
		m.ns = nsGetter.GetNS()
	}
	if schemaGetter, ok := v.(SchemaGetter); ok {
		m.schema = schemaGetter.GetSchema()
	}

	var hasPrimaryKey bool
	if idGetter, ok := v.(IDGetter); ok {
		m.id = idGetter.GetID()
		hasPrimaryKey = true
	} else {
		for _, f := range fields {
			if f.primaryKey {
				hasPrimaryKey = true
				strVal, ok := f.value.(string)
				if !ok {
					return m, ErrPrimaryKeyNotString
				}
				m.id = strVal
				break
			}
		}
	}

	if !hasPrimaryKey {
		return m, ErrNoPrimaryKey
	}

	return m, m.validate()
}

func (e *Encoder) buildRecord(meta metadata, fields []field) *core.Record {
	record := core.NewRecord(meta.ns, meta.schema, meta.id)

	for _, f := range fields {
		if f.skip || f.primaryKey || f.value == nil {
			continue
		}
		record.Set(f.name, f.value)
	}

	return record
}

func (e *Encoder) parseFields(rv reflect.Value, prefix string) ([]field, error) {
	rt := rv.Type()
	var fields []field

	for i := 0; i < rv.NumField(); i++ {
		structField := rt.Field(i)
		fieldValue := rv.Field(i)

		if !structField.IsExported() {
			continue
		}

		f := parseTag(structField.Tag.Get(e.opts.Tag))
		if f.skip {
			continue
		}

		if f.name == "" {
			f.name = structField.Name
		}

		if prefix != "" {
			f.name = prefix + "." + f.name
		}

		for fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				break
			}
			fieldValue = fieldValue.Elem()
		}

		if fieldValue.Kind() == reflect.Struct {
			nestedPrefix := prefix
			if !structField.Anonymous {
				nestedPrefix = f.name
			}
			nestedFields, err := e.parseFields(fieldValue, nestedPrefix)
			if err != nil {
				return nil, err
			}
			fields = append(fields, nestedFields...)
			continue
		}

		value, err := e.getValue(fieldValue)
		if err != nil {
			return nil, err
		}
		f.value = value

		fields = append(fields, f)
	}

	return fields, nil
}

func (e *Encoder) getValue(rv reflect.Value) (any, error) {
	rv = e.derefPointer(rv)
	if !rv.IsValid() {
		return nil, nil
	}

	if e.implementsMarshaler(rv) {
		return e.marshalField(rv)
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return e.encodeSlice(rv)
	default:
		return rv.Interface(), nil
	}
}

func (e *Encoder) derefPointer(rv reflect.Value) reflect.Value {
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

func (e *Encoder) encodeSlice(rv reflect.Value) (any, error) {
	if rv.IsNil() || rv.Len() == 0 {
		return nil, nil
	}

	if rv.Type().Elem().Kind() == reflect.Uint8 {
		return rv.Bytes(), nil
	}

	values := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		val, err := e.encodeElement(rv.Index(i))
		if err != nil {
			return nil, err
		}
		values[i] = val
	}

	return values, nil
}

func (e *Encoder) encodeElement(rv reflect.Value) (any, error) {
	rv = e.derefPointer(rv)
	if !rv.IsValid() {
		return nil, nil
	}

	if e.implementsMarshaler(rv) {
		return e.marshalField(rv)
	}

	return rv.Interface(), nil
}

func (e *Encoder) implementsMarshaler(rv reflect.Value) bool {
	if !rv.IsValid() || !rv.CanInterface() {
		return false
	}

	iface := rv.Interface()
	_, isJSON := iface.(json.Marshaler)
	_, isBinary := iface.(encoding.BinaryMarshaler)
	return isJSON || isBinary
}

func (e *Encoder) marshalField(rv reflect.Value) ([]byte, error) {
	iface := rv.Interface()

	if m, ok := iface.(json.Marshaler); ok {
		return m.MarshalJSON()
	}

	if m, ok := iface.(encoding.BinaryMarshaler); ok {
		return m.MarshalBinary()
	}

	return nil, ErrNoMarshaler
}

type metadata struct {
	ns     string
	schema string
	id     string
}

func (m metadata) validate() error {
	if m.ns == "" {
		return ErrEmptyNamespace
	}
	if m.schema == "" {
		return ErrEmptySchema
	}
	if m.id == "" {
		return ErrEmptyID
	}
	return nil
}
