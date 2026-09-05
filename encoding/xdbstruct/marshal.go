package xdbstruct

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Marshal encodes v into a [core.Record] addressed by uri. The uri must carry
// ns, schema, and id. If one of them is missing, Marshal panics. v must be a
// struct or a pointer to one, of the type that produced the schema.
//
// Nested structs become dotted attributes; []Struct becomes an ARRAY<JSON>
// object array whose elements are typed JSON (time.Time as RFC3339, []byte as
// base64) so they round-trip through [Unmarshal]. A nil pointer field is absent
// (no tuple), preserving the absent-vs-null distinction.
func Marshal(uri string, v any) (*core.Record, error) {
	u, err := core.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	specs, err := buildStruct(rv.Type(), []reflect.Type{rv.Type()})
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(u.NS(), u.Schema(), u.ID())
	if err := encodeInto(record, specs, rv, ""); err != nil {
		return nil, err
	}

	return record, nil
}

// encodeInto writes tuples for specs from structVal into record. prefix carries
// the dotted namespace for nested structs.
func encodeInto(record *core.Record, specs []spec, structVal reflect.Value, prefix string) error {
	for _, s := range specs {
		fv, ok := fieldByIndex(structVal, s.index)
		if !ok {
			continue // absent through a nil embedded pointer
		}
		if s.ptr {
			if fv.IsNil() {
				continue // nil pointer → absent
			}
			fv = fv.Elem()
		}

		switch s.kind {
		case kindNested:
			if err := encodeInto(record, s.children, fv, prefix+s.attr+"."); err != nil {
				return err
			}

		case kindScalarArray:
			if fv.IsNil() {
				continue
			}
			v, err := scalarArrayValue(fv, s)
			if err != nil {
				return err
			}
			record.Set(prefix+s.attr, v)

		case kindObjectArray:
			if fv.IsNil() {
				continue
			}
			v, err := objectArrayValue(fv, s)
			if err != nil {
				return err
			}
			record.Set(prefix+s.attr, v)

		default: // kindLeaf
			v, err := leafValue(fv, s.coreType.ID(), s.goType)
			if err != nil {
				return err
			}
			record.Set(prefix+s.attr, v)
		}
	}

	return nil
}

// leafValue builds a scalar/time/bytes/JSON [core.Value] from a field value.
func leafValue(fv reflect.Value, tid core.TID, goType reflect.Type) (*core.Value, error) {
	switch tid {
	case core.TIDTime:
		return core.TimeVal(fv.Interface().(time.Time)), nil
	case core.TIDBytes:
		return core.BytesVal(fv.Bytes()), nil
	case core.TIDJSON:
		if goType == rawMsgType {
			return core.JSONVal(append(json.RawMessage(nil), fv.Bytes()...)), nil
		}
		raw, err := json.Marshal(fv.Interface())
		if err != nil {
			return nil, err
		}
		return core.JSONVal(raw), nil
	case core.TIDBoolean:
		return core.BoolVal(fv.Bool()), nil
	case core.TIDInteger:
		return core.IntVal(fv.Int()), nil
	case core.TIDUnsigned:
		return core.UintVal(fv.Uint()), nil
	case core.TIDFloat:
		return core.FloatVal(fv.Float()), nil
	default: // core.TIDString
		return core.StringVal(fv.String()), nil
	}
}

// scalarArrayValue builds an ARRAY<scalar> value from a slice field.
func scalarArrayValue(fv reflect.Value, s spec) (*core.Value, error) {
	tid := s.coreType.ElemTypeID()
	elems := make([]*core.Value, 0, fv.Len())

	for i := range fv.Len() {
		ev := fv.Index(i)
		if s.elemPtr {
			if ev.IsNil() {
				elems = append(elems, nil)
				continue
			}
			ev = ev.Elem()
		}
		v, err := leafValue(ev, tid, s.elemType)
		if err != nil {
			return nil, err
		}
		elems = append(elems, v)
	}

	return core.ArrayVal(tid, elems...), nil
}

// objectArrayValue builds an ARRAY<JSON> value from a slice of structs, encoding
// each element as a typed JSON object.
func objectArrayValue(fv reflect.Value, s spec) (*core.Value, error) {
	elems := make([]*core.Value, 0, fv.Len())

	for i := range fv.Len() {
		ev := fv.Index(i)
		if s.elemPtr {
			if ev.IsNil() {
				elems = append(elems, nil)
				continue
			}
			ev = ev.Elem()
		}
		raw, err := json.Marshal(encodeElement(s.children, ev))
		if err != nil {
			return nil, err
		}
		elems = append(elems, core.JSONVal(raw))
	}

	return core.ArrayVal(core.TIDJSON, elems...), nil
}

// encodeElement renders one object-array element as a map ready for
// json.Marshal. Leaf and scalar-array members use the Go value directly so
// json's own encoding types them (time.Time→RFC3339, []byte→base64); nested
// structs flatten to dotted keys; nested object arrays recurse.
func encodeElement(specs []spec, structVal reflect.Value) map[string]any {
	m := make(map[string]any)

	for _, s := range specs {
		fv, ok := fieldByIndex(structVal, s.index)
		if !ok {
			continue
		}
		if s.ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		switch s.kind {
		case kindNested:
			for k, v := range encodeElement(s.children, fv) {
				m[s.attr+"."+k] = v
			}

		case kindObjectArray:
			if fv.IsNil() {
				continue
			}
			arr := make([]any, 0, fv.Len())
			for i := range fv.Len() {
				ev := fv.Index(i)
				if s.elemPtr {
					if ev.IsNil() {
						arr = append(arr, nil)
						continue
					}
					ev = ev.Elem()
				}
				arr = append(arr, encodeElement(s.children, ev))
			}
			m[s.attr] = arr

		default: // kindLeaf, kindScalarArray
			if (s.kind == kindScalarArray) && fv.IsNil() {
				continue
			}
			m[s.attr] = fv.Interface()
		}
	}

	return m
}

// fieldByIndex resolves an index path, dereferencing embedded pointers. It
// reports false when a nil embedded pointer makes the field unreachable.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, bool) {
	for i, x := range index {
		if i > 0 {
			for v.Kind() == reflect.Pointer {
				if v.IsNil() {
					return reflect.Value{}, false
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v, true
}
