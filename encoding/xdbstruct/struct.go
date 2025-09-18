// Package xdbstruct provides utilities for converting Go structs to XDB records and vice versa.
package xdbstruct

import (
	"encoding"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	// ErrNotStruct is returned when ToRecord is called with a non-struct argument.
	ErrNotStruct = errors.New("encoding/xdbstruct: ToRecord expects a pointer to a struct")
)

// ToRecord converts a struct to a core.Record.
func ToRecord(obj any) (*core.Record, error) {
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	tuples, err := marshalStruct(v, nil)
	if err != nil {
		return nil, err
	}

	var kind, id string

	kind = v.Type().Name()

	for _, v := range tuples {
		if v.PrimaryKey {
			id = v.Value.(string)
		}
	}

	record := core.NewRecord(kind, id)

	for _, v := range tuples {
		record.Set(v.Name, v.Value)
	}

	return record, nil
}

// FromRecord converts a core.Record to a struct.
func FromRecord(record *core.Record, obj any) error {
	return nil
}

func marshalStruct(v reflect.Value, parent *meta) (map[string]*meta, error) {
	typ := v.Type()
	allTuples := make(map[string]*meta)

	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		fieldValue := v.Field(i)

		meta, err := parseTag(fieldType)
		if err != nil {
			return nil, err
		}

		if meta == nil {
			continue
		}

		if parent != nil {
			meta.Name = parent.Name + "." + meta.Name
			// nested primary keys are not supported
			meta.PrimaryKey = false
		}

		parsed, err := processField(fieldValue, meta)
		if err != nil {
			return nil, errors.Wrap(err, "field", meta.Name)
		}

		for k, v := range parsed {
			allTuples[k] = v
		}
	}

	return allTuples, nil
}

func processField(fieldValue reflect.Value, m *meta) (map[string]*meta, error) {
	parsed := make(map[string]*meta)

	// keep time.Time as is
	if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
		m.Value = fieldValue.Interface()
		parsed[m.Name] = m

		return parsed, nil
	}

	// try handling custom marshalers
	binary, err := tryCustomMarshal(fieldValue)
	if err != nil {
		return nil, errors.Wrap(err, "field", m.Name)
	}

	if binary != nil {
		m.Value = binary
		parsed[m.Name] = m

		return parsed, nil
	}

	// recursively marshal slices and arrays of structs
	if isBinaryArray(fieldValue) {
		array := make([][]byte, fieldValue.Len())
		for i := 0; i < fieldValue.Len(); i++ {
			binary, err := tryCustomMarshal(fieldValue.Index(i))
			if err != nil {
				return nil, errors.Wrap(err, "field", m.Name)
			}

			array[i] = binary
		}

		m.Value = array
		parsed[m.Name] = m

		return parsed, nil
	}

	// flatten nested structs
	if fieldValue.Kind() == reflect.Struct {
		nested, err := marshalStruct(fieldValue, m)
		if err != nil {
			return nil, errors.Wrap(err, "field", m.Name)
		}

		for k, v := range nested {
			parsed[k] = v
		}

		return parsed, nil
	}

	m.Value = fieldValue.Interface()
	parsed[m.Name] = m

	return parsed, nil
}

func tryCustomMarshal(v reflect.Value) ([]byte, error) {
	if jm, ok := v.Interface().(json.Marshaler); ok {
		return jm.MarshalJSON()
	}

	if bm, ok := v.Interface().(encoding.BinaryMarshaler); ok {
		return bm.MarshalBinary()
	}

	return nil, nil
}

func isBinaryArray(v reflect.Value) bool {
	isArray := v.Kind() == reflect.Slice || v.Kind() == reflect.Array

	if !isArray {
		return false
	}

	elem := v.Type().Elem()

	// if it's a slice of time.Time, it's not a binary array
	if elem.PkgPath() == "time" && elem.Name() == "Time" {
		return false
	}

	if _, ok := elem.MethodByName("MarshalJSON"); ok {
		return true
	}

	if _, ok := elem.MethodByName("MarshalBinary"); ok {
		return true
	}

	return false
}

type meta struct {
	Name       string
	PrimaryKey bool
	Value      any
	Options    map[string]string
}

func parseTag(field reflect.StructField) (*meta, error) {
	tag := field.Tag.Get("xdb")
	if tag == "" || tag == "-" {
		return nil, nil
	}

	// skip unexported fields
	if field.PkgPath != "" {
		return nil, nil
	}

	parts := strings.Split(tag, ",")
	key, tagOpts := strings.TrimSpace(parts[0]), parts[1:]

	meta := &meta{
		Name:       key,
		PrimaryKey: strings.Contains(key, "primary_key"),
		Options:    make(map[string]string),
	}

	for _, part := range tagOpts {
		kv := strings.Split(part, "=")

		if kv[0] == "primary_key" {
			meta.PrimaryKey = true
		}

		if len(kv) == 1 {
			meta.Options[kv[0]] = ""
		} else {
			meta.Options[kv[0]] = kv[1]
		}
	}

	return meta, nil
}
