package xdbstruct

import (
	"encoding"
	"encoding/json"
	"reflect"

	"github.com/xdb-dev/xdb/core"
)

// Decoder converts XDB records to Go structs.
type Decoder struct {
	opts Options
}

// NewDecoder creates a decoder with custom options.
func NewDecoder(opts Options) *Decoder {
	if opts.Tag == "" {
		opts.Tag = "xdb"
	}
	return &Decoder{opts: opts}
}

// NewDefaultDecoder creates a decoder with default options (Tag: "xdb").
func NewDefaultDecoder() *Decoder {
	return NewDecoder(DefaultOptions())
}

// FromRecord populates a struct from a core.Record.
// Returns an error if:
//   - v is not a pointer to struct
//   - Type mismatch between record value and struct field
//   - Custom unmarshaler fails
func (d *Decoder) FromRecord(record *core.Record, v any) error {
	rv, err := d.validateInput(v)
	if err != nil {
		return err
	}

	fieldIndex := d.buildFieldIndex(rv.Type(), "", nil)

	if err := d.populateFields(rv, record, fieldIndex); err != nil {
		return err
	}

	d.callSetters(v, record)

	return nil
}

func (d *Decoder) validateInput(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return rv, ErrInvalidInput
	}

	if rv.Kind() != reflect.Ptr {
		return rv, ErrNotPointer
	}

	if rv.IsNil() {
		return rv, ErrNilPointer
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return rv, ErrNotStruct
	}

	return rv, nil
}

type fieldInfo struct {
	indices    []int
	primaryKey bool
}

func (d *Decoder) buildFieldIndex(rt reflect.Type, prefix string, parentIndices []int) map[string]fieldInfo {
	index := make(map[string]fieldInfo)

	for i := 0; i < rt.NumField(); i++ {
		structField := rt.Field(i)
		if !structField.IsExported() {
			continue
		}

		f := parseTag(structField.Tag.Get(d.opts.Tag))
		if f.skip {
			continue
		}

		fieldName := f.name
		if fieldName == "" {
			fieldName = structField.Name
		}

		fullName := fieldName
		if prefix != "" {
			fullName = prefix + "." + fieldName
		}

		indices := append(append([]int{}, parentIndices...), i)

		fieldType := structField.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct && !d.isSpecialType(fieldType) {
			nestedPrefix := prefix
			if !structField.Anonymous {
				nestedPrefix = fullName
			}
			nestedIndex := d.buildFieldIndex(fieldType, nestedPrefix, indices)
			for k, v := range nestedIndex {
				index[k] = v
			}
		} else {
			index[fullName] = fieldInfo{
				indices:    indices,
				primaryKey: f.primaryKey,
			}
		}
	}

	return index
}

func (d *Decoder) isSpecialType(rt reflect.Type) bool {
	iface := reflect.New(rt).Interface()
	if _, ok := iface.(json.Unmarshaler); ok {
		return true
	}
	if _, ok := iface.(encoding.BinaryUnmarshaler); ok {
		return true
	}
	return false
}

func (d *Decoder) populateFields(rv reflect.Value, record *core.Record, fieldIndex map[string]fieldInfo) error {
	for attrName, fi := range fieldIndex {
		if fi.primaryKey {
			id := core.NewValue(record.ID().String())
			if err := d.setFieldByIndices(rv, fi.indices, id); err != nil {
				return err
			}
			continue
		}

		tuple := record.Get(attrName)
		if tuple == nil {
			continue
		}

		value := tuple.Value()
		if value == nil || value.IsNil() {
			continue
		}

		if err := d.setFieldByIndices(rv, fi.indices, value); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) setFieldByIndices(rv reflect.Value, indices []int, value *core.Value) error {
	target := rv
	for i, idx := range indices {
		if target.Kind() == reflect.Ptr {
			if target.IsNil() {
				target.Set(reflect.New(target.Type().Elem()))
			}
			target = target.Elem()
		}

		if i < len(indices)-1 {
			target = target.Field(idx)
			if target.Kind() == reflect.Ptr {
				if target.IsNil() {
					target.Set(reflect.New(target.Type().Elem()))
				}
				target = target.Elem()
			}
		} else {
			target = target.Field(idx)
		}
	}

	return d.setValue(target, value)
}

func (d *Decoder) setValue(target reflect.Value, value *core.Value) error {
	if target.Kind() == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}

	targetType := target.Type()

	if targetType == reflect.TypeOf(json.RawMessage{}) {
		bytes := value.ToBytes()
		target.SetBytes(bytes)
		return nil
	}

	typeID := value.Type().ID()

	switch target.Kind() {
	case reflect.Bool:
		if !isConvertibleToBool(typeID) {
			return ErrTypeMismatch
		}
		target.SetBool(value.ToBool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !isConvertibleToInt(typeID) {
			return ErrTypeMismatch
		}
		target.SetInt(value.ToInt())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !isConvertibleToUint(typeID) {
			return ErrTypeMismatch
		}
		target.SetUint(value.ToUint())
	case reflect.Float32, reflect.Float64:
		if !isConvertibleToFloat(typeID) {
			return ErrTypeMismatch
		}
		target.SetFloat(value.ToFloat())
	case reflect.String:
		target.SetString(value.ToString())
	case reflect.Slice:
		return d.setSliceValue(target, value)
	default:
		return ErrTypeMismatch
	}

	return nil
}

func isConvertibleToBool(typeID core.TID) bool {
	switch typeID {
	case core.TIDBoolean, core.TIDInteger, core.TIDUnsigned, core.TIDFloat:
		return true
	default:
		return false
	}
}

func isConvertibleToInt(typeID core.TID) bool {
	switch typeID {
	case core.TIDBoolean, core.TIDInteger, core.TIDUnsigned, core.TIDFloat:
		return true
	default:
		return false
	}
}

func isConvertibleToUint(typeID core.TID) bool {
	switch typeID {
	case core.TIDBoolean, core.TIDInteger, core.TIDUnsigned, core.TIDFloat:
		return true
	default:
		return false
	}
}

func isConvertibleToFloat(typeID core.TID) bool {
	switch typeID {
	case core.TIDBoolean, core.TIDInteger, core.TIDUnsigned, core.TIDFloat:
		return true
	default:
		return false
	}
}

func (d *Decoder) setSliceValue(target reflect.Value, value *core.Value) error {
	elemKind := target.Type().Elem().Kind()

	if elemKind == reflect.Uint8 {
		target.SetBytes(value.ToBytes())
		return nil
	}

	switch elemKind {
	case reflect.Bool:
		arr := value.ToBoolArray()
		target.Set(reflect.ValueOf(arr))
	case reflect.Int:
		arr := value.ToIntArray()
		result := make([]int, len(arr))
		for i, v := range arr {
			result[i] = int(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Int8:
		arr := value.ToIntArray()
		result := make([]int8, len(arr))
		for i, v := range arr {
			result[i] = int8(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Int16:
		arr := value.ToIntArray()
		result := make([]int16, len(arr))
		for i, v := range arr {
			result[i] = int16(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Int32:
		arr := value.ToIntArray()
		result := make([]int32, len(arr))
		for i, v := range arr {
			result[i] = int32(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Int64:
		target.Set(reflect.ValueOf(value.ToIntArray()))
	case reflect.Uint:
		arr := value.ToUintArray()
		result := make([]uint, len(arr))
		for i, v := range arr {
			result[i] = uint(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Uint16:
		arr := value.ToUintArray()
		result := make([]uint16, len(arr))
		for i, v := range arr {
			result[i] = uint16(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Uint32:
		arr := value.ToUintArray()
		result := make([]uint32, len(arr))
		for i, v := range arr {
			result[i] = uint32(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Uint64:
		target.Set(reflect.ValueOf(value.ToUintArray()))
	case reflect.Float32:
		arr := value.ToFloatArray()
		result := make([]float32, len(arr))
		for i, v := range arr {
			result[i] = float32(v)
		}
		target.Set(reflect.ValueOf(result))
	case reflect.Float64:
		target.Set(reflect.ValueOf(value.ToFloatArray()))
	case reflect.String:
		target.Set(reflect.ValueOf(value.ToStringArray()))
	default:
		return ErrTypeMismatch
	}

	return nil
}

func (d *Decoder) callSetters(v any, record *core.Record) {
	if setter, ok := v.(IDSetter); ok {
		setter.SetID(record.ID().String())
	}

	if setter, ok := v.(NSSetter); ok {
		ns := record.NS()
		if ns != nil {
			setter.SetNS(ns.String())
		}
	}

	if setter, ok := v.(SchemaSetter); ok {
		schema := record.Schema()
		if schema != nil {
			setter.SetSchema(schema.String())
		}
	}
}
