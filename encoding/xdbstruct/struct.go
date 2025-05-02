package xdbstruct

import (
	"encoding"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/xdb-dev/xdb/types"
)

var (
	ErrNotStruct = errors.New("xdbstruct: ToRecord expects a pointer to a struct")
)

// ToRecord converts a struct to a types.Record.
func ToRecord(obj any) (*types.Record, error) {
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

	record := types.NewRecord(kind, id)

	for _, v := range tuples {
		record.Set(v.Name, v.Value)
	}

	return record, nil
}

// FromRecord converts a types.Record to a struct.
func FromRecord(record *types.Record, obj any) error {
	return nil
}

func marshalStruct(v reflect.Value, parent *field) (map[string]*field, error) {
	typ := v.Type()
	tuples := make(map[string]*field)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := v.Field(i)

		tag := field.Tag.Get("xdb")
		if tag == "" || tag == "-" {
			continue
		}

		// skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		tuple, err := parseTag(tag)
		if err != nil {
			return nil, err
		}

		if parent != nil {
			tuple.Name = parent.Name + "." + tuple.Name
			// nested primary keys are not supported
			tuple.PrimaryKey = false
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			switch fieldValue.Interface().(type) {
			case json.Marshaler:
				tuple.Value, err = fieldValue.Interface().(json.Marshaler).MarshalJSON()
				if err != nil {
					return nil, err
				}
			case encoding.BinaryMarshaler:
				tuple.Value, err = fieldValue.Interface().(encoding.BinaryMarshaler).MarshalBinary()
				if err != nil {
					return nil, err
				}
			default:
				nested, err := marshalStruct(fieldValue, tuple)
				if err != nil {
					return nil, err
				}

				for k, v := range nested {
					tuples[k] = v
				}
			}

		default:
			tuple.Value = fieldValue.Interface()
			tuples[tuple.Name] = tuple
		}
	}

	return tuples, nil
}

type field struct {
	Name       string
	PrimaryKey bool
	Value      any
	Options    map[string]string
}

func parseTag(tag string) (*field, error) {
	parts := strings.Split(tag, ",")
	key, tagOpts := strings.TrimSpace(parts[0]), parts[1:]

	meta := &field{
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
