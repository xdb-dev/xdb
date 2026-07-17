package xdbstruct

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/xdb-dev/xdb/core"
)

// Unmarshal decodes rec into dst, which must be a non-nil pointer to a struct of
// the type that produced the schema.
//
// It reverses [Marshal]: dotted attributes rebuild nested structs (a nil
// pointer when the whole subtree is absent), object arrays decode back into
// []Struct, and named scalar types are restored through the destination's
// static type.
func Unmarshal(rec *core.Record, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return ErrNotStruct
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	specs, err := buildStruct(elem.Type(), []reflect.Type{elem.Type()})
	if err != nil {
		return err
	}

	return decodeInto(rec, specs, elem, "")
}

// decodeInto assigns specs from record into structVal. prefix carries the dotted
// namespace for nested structs.
func decodeInto(rec *core.Record, specs []spec, structVal reflect.Value, prefix string) error {
	for _, s := range specs {
		switch s.kind {
		case kindNested:
			childPrefix := prefix + s.attr + "."
			if !hasPrefix(rec, childPrefix) {
				continue // whole subtree absent → leave nil/zero
			}
			fv := settableFieldByIndex(structVal, s.index)
			if s.ptr {
				nv := reflect.New(s.goType)
				if err := decodeInto(rec, s.children, nv.Elem(), childPrefix); err != nil {
					return err
				}
				fv.Set(nv)
			} else if err := decodeInto(rec, s.children, fv, childPrefix); err != nil {
				return err
			}

		default:
			tuple := rec.Get(prefix + s.attr)
			if tuple == nil {
				continue // absent → leave zero/nil
			}
			fv := settableFieldByIndex(structVal, s.index)
			if err := decodeField(fv, s, tuple.Value()); err != nil {
				return err
			}
		}
	}

	return nil
}

// decodeField assigns a leaf/scalar-array/object-array value into fv, allocating
// a pointer target when the field is nullable.
func decodeField(fv reflect.Value, s spec, val *core.Value) error {
	target := fv
	if s.ptr {
		target = reflect.New(s.goType).Elem()
	}

	var err error
	switch s.kind {
	case kindScalarArray:
		err = decodeScalarArray(target, s, val)
	case kindObjectArray:
		err = decodeObjectArray(target, s, val)
	default: // kindLeaf
		err = assignValue(target, s.coreType.ID(), s.goType, val)
	}
	if err != nil {
		return err
	}

	if s.ptr {
		p := reflect.New(s.goType)
		p.Elem().Set(target)
		fv.Set(p)
	}
	return nil
}

// assignValue sets target from a leaf [core.Value] according to its type.
func assignValue(target reflect.Value, tid core.TID, goType reflect.Type, val *core.Value) error {
	switch tid {
	case core.TIDTime:
		t, err := val.AsTime()
		if err != nil {
			return err
		}
		target.Set(reflect.ValueOf(t))
		return nil
	case core.TIDBytes:
		b, err := val.AsBytes()
		if err != nil {
			return err
		}
		target.SetBytes(b)
		return nil
	case core.TIDJSON:
		return assignJSON(target, goType, val)
	default:
		return assignScalar(target, tid, val)
	}
}

// assignJSON sets a JSON target: a json.RawMessage receives the raw bytes;
// anything else (a json-opt-in map or interface) is unmarshalled into.
func assignJSON(target reflect.Value, goType reflect.Type, val *core.Value) error {
	raw, err := val.AsJSON()
	if err != nil {
		return err
	}
	if goType == rawMsgType {
		target.SetBytes(append(json.RawMessage(nil), raw...))
		return nil
	}
	return json.Unmarshal(raw, target.Addr().Interface())
}

// assignScalar sets a bool/int/uint/float/string target from val.
func assignScalar(target reflect.Value, tid core.TID, val *core.Value) error {
	switch tid {
	case core.TIDBoolean:
		b, err := val.AsBool()
		if err != nil {
			return err
		}
		target.SetBool(b)
	case core.TIDInteger:
		i, err := val.AsInt()
		if err != nil {
			return err
		}
		target.SetInt(i)
	case core.TIDUnsigned:
		u, err := val.AsUint()
		if err != nil {
			return err
		}
		target.SetUint(u)
	case core.TIDFloat:
		f, err := val.AsFloat()
		if err != nil {
			return err
		}
		target.SetFloat(f)
	default: // core.TIDString
		str, err := val.AsStr()
		if err != nil {
			return err
		}
		target.SetString(str)
	}
	return nil
}

// decodeScalarArray sets target (a slice) from an ARRAY<scalar> value.
func decodeScalarArray(target reflect.Value, s spec, val *core.Value) error {
	elems, err := val.AsArray()
	if err != nil {
		return err
	}

	tid := s.coreType.ElemTypeID()
	slice := reflect.MakeSlice(target.Type(), len(elems), len(elems))

	for i, ev := range elems {
		if ev == nil {
			continue // nil element → zero/nil
		}
		out := reflect.New(s.elemType).Elem()
		if err := assignValue(out, tid, s.elemType, ev); err != nil {
			return err
		}
		if s.elemPtr {
			p := reflect.New(s.elemType)
			p.Elem().Set(out)
			slice.Index(i).Set(p)
		} else {
			slice.Index(i).Set(out)
		}
	}

	target.Set(slice)
	return nil
}

// decodeObjectArray sets target (a slice of structs) from an ARRAY<JSON> value.
func decodeObjectArray(target reflect.Value, s spec, val *core.Value) error {
	elems, err := val.AsArray()
	if err != nil {
		return err
	}

	slice := reflect.MakeSlice(target.Type(), len(elems), len(elems))
	for i, ev := range elems {
		if ev == nil {
			continue
		}
		raw, err := ev.AsJSON()
		if err != nil {
			return err
		}
		out := reflect.New(s.elemType).Elem()
		if err := decodeElement(s.children, raw, out); err != nil {
			return err
		}
		if s.elemPtr {
			p := reflect.New(s.elemType)
			p.Elem().Set(out)
			slice.Index(i).Set(p)
		} else {
			slice.Index(i).Set(out)
		}
	}

	target.Set(slice)
	return nil
}

// decodeElement decodes one typed-JSON object-array element into structVal,
// mirroring encodeElement.
func decodeElement(specs []spec, raw json.RawMessage, structVal reflect.Value) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}

	for _, s := range specs {
		var err error
		switch s.kind {
		case kindNested:
			err = decodeElemNested(s, obj, structVal)
		case kindObjectArray:
			err = decodeElemObjectArray(s, obj, structVal)
		default: // kindLeaf, kindScalarArray
			err = decodeElemLeaf(s, obj, structVal)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// decodeElemNested rebuilds a nested struct member from dotted-prefixed keys.
func decodeElemNested(s spec, obj map[string]json.RawMessage, structVal reflect.Value) error {
	sub := make(map[string]json.RawMessage)
	p := s.attr + "."
	for k, v := range obj {
		if strings.HasPrefix(k, p) {
			sub[k[len(p):]] = v
		}
	}
	if len(sub) == 0 {
		return nil
	}

	subRaw, err := json.Marshal(sub)
	if err != nil {
		return err
	}

	fv := settableFieldByIndex(structVal, s.index)
	if s.ptr {
		nv := reflect.New(s.goType)
		if err := decodeElement(s.children, subRaw, nv.Elem()); err != nil {
			return err
		}
		fv.Set(nv)
		return nil
	}
	return decodeElement(s.children, subRaw, fv)
}

// decodeElemObjectArray decodes a nested object-array member.
func decodeElemObjectArray(s spec, obj map[string]json.RawMessage, structVal reflect.Value) error {
	mr, ok := obj[s.attr]
	if !ok || isJSONNull(mr) {
		return nil
	}

	var rawElems []json.RawMessage
	if err := json.Unmarshal(mr, &rawElems); err != nil {
		return err
	}

	fv := settableFieldByIndex(structVal, s.index)
	slice := reflect.MakeSlice(fv.Type(), len(rawElems), len(rawElems))
	for i, re := range rawElems {
		out := reflect.New(s.elemType).Elem()
		if err := decodeElement(s.children, re, out); err != nil {
			return err
		}
		setSliceElem(slice.Index(i), out, s.elemType, s.elemPtr)
	}
	fv.Set(slice)
	return nil
}

// decodeElemLeaf decodes a leaf or scalar-array member via encoding/json, which
// natively handles time.Time, []byte, named types, and json-opt-in values.
func decodeElemLeaf(s spec, obj map[string]json.RawMessage, structVal reflect.Value) error {
	mr, ok := obj[s.attr]
	if !ok || isJSONNull(mr) {
		return nil
	}

	fv := settableFieldByIndex(structVal, s.index)
	ptr := s.kind != kindScalarArray && s.ptr

	target := fv
	if ptr {
		target = reflect.New(s.goType).Elem()
	}
	if err := json.Unmarshal(mr, target.Addr().Interface()); err != nil {
		return err
	}
	if ptr {
		p := reflect.New(s.goType)
		p.Elem().Set(target)
		fv.Set(p)
	}
	return nil
}

// setSliceElem sets slot to val, boxing it in a pointer when the slice element
// type is a pointer.
func setSliceElem(slot, val reflect.Value, elemType reflect.Type, elemPtr bool) {
	if elemPtr {
		p := reflect.New(elemType)
		p.Elem().Set(val)
		slot.Set(p)
		return
	}
	slot.Set(val)
}

// settableFieldByIndex resolves an index path, allocating nil embedded pointers
// so the returned value is settable.
func settableFieldByIndex(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			for v.Kind() == reflect.Pointer {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}

// hasPrefix reports whether any tuple attribute starts with prefix.
func hasPrefix(rec *core.Record, prefix string) bool {
	for _, t := range rec.Tuples() {
		if strings.HasPrefix(t.Attr(), prefix) {
			return true
		}
	}
	return false
}

// isJSONNull reports whether raw is the JSON null literal.
func isJSONNull(raw json.RawMessage) bool {
	return string(raw) == "null"
}
