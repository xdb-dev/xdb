package protoimport

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/xdb-dev/xdb/core"
)

// Unmarshal decodes rec into msg, which must be a mutable proto message of the
// type that produced the schema (e.g. a fresh dynamicpb.Message).
//
// It reverses [Marshal]: dotted attributes rebuild nested messages, object
// arrays decode back into repeated messages, enums are matched by value name,
// and Timestamp/bytes are restored from their typed values.
func Unmarshal(rec *core.Record, msg proto.Message, opts ...Option) error {
	o := buildOptions(opts)
	m := msg.ProtoReflect()
	md := m.Descriptor()

	plan, err := buildMessagePlan(md, []protoreflect.FullName{md.FullName()}, o.allowJSON)
	if err != nil {
		return err
	}

	return decodeInto(rec, plan, m, "")
}

// decodeInto assigns plan from rec into m. prefix carries the dotted namespace
// for nested messages.
func decodeInto(rec *core.Record, plan []fieldPlan, m protoreflect.Message, prefix string) error {
	for _, fp := range plan {
		if fp.kind == kindNested {
			childPrefix := prefix + fp.attr + "."
			if !hasPrefix(rec, childPrefix) {
				continue // whole subtree absent
			}
			sub := m.Mutable(fp.fd).Message()
			if err := decodeInto(rec, fp.items, sub, childPrefix); err != nil {
				return err
			}
			continue
		}

		tuple := rec.Get(prefix + fp.attr)
		if tuple == nil {
			continue // absent → leave unset
		}
		if err := decodeField(m, fp, tuple.Value()); err != nil {
			return err
		}
	}

	return nil
}

// decodeField assigns a leaf/scalar-array/object-array value into m's field.
func decodeField(m protoreflect.Message, fp fieldPlan, val *core.Value) error {
	switch fp.kind {
	case kindScalarArray:
		return decodeScalarArray(m, fp, val)
	case kindObjectArray:
		return decodeObjectArray(m, fp, val)
	default: // kindLeaf
		return decodeLeaf(m, fp, val)
	}
}

// decodeLeaf sets a single leaf value onto m's field.
func decodeLeaf(m protoreflect.Message, fp fieldPlan, val *core.Value) error {
	switch {
	case fp.cat == catJSON:
		return decodeJSON(m, fp, val)

	case fp.wrapped:
		sub := m.Mutable(fp.fd).Message()
		valueFd := sub.Descriptor().Fields().ByName("value")
		pv, err := scalarProtoValue(valueFd, fp.cat, val)
		if err != nil {
			return err
		}
		sub.Set(valueFd, pv)
		return nil

	case fp.cat == catTime:
		return decodeTime(m, fp.fd, val)

	case fp.cat == catEnum:
		return decodeEnum(m, fp.fd, val)

	default:
		pv, err := scalarProtoValue(fp.fd, fp.cat, val)
		if err != nil {
			return err
		}
		m.Set(fp.fd, pv)
		return nil
	}
}

// scalarProtoValue builds a scalar [protoreflect.Value] matching fd's width.
func scalarProtoValue(fd protoreflect.FieldDescriptor, cat leafCat, val *core.Value) (protoreflect.Value, error) {
	switch cat {
	case catBool:
		b, err := val.AsBool()
		return protoreflect.ValueOfBool(b), err
	case catInt:
		i, err := val.AsInt()
		return intProtoValue(fd, i), err
	case catUint:
		u, err := val.AsUint()
		return uintProtoValue(fd, u), err
	case catFloat:
		f, err := val.AsFloat()
		return floatProtoValue(fd, f), err
	case catBytes:
		b, err := val.AsBytes()
		return protoreflect.ValueOfBytes(b), err
	default: // catString
		s, err := val.AsStr()
		return protoreflect.ValueOfString(s), err
	}
}

// intProtoValue selects the 32- or 64-bit int representation for fd.
func intProtoValue(fd protoreflect.FieldDescriptor, i int64) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(i))
	default:
		return protoreflect.ValueOfInt64(i)
	}
}

// uintProtoValue selects the 32- or 64-bit uint representation for fd.
func uintProtoValue(fd protoreflect.FieldDescriptor, u uint64) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(u))
	default:
		return protoreflect.ValueOfUint64(u)
	}
}

// floatProtoValue selects the 32- or 64-bit float representation for fd.
func floatProtoValue(fd protoreflect.FieldDescriptor, f float64) protoreflect.Value {
	if fd.Kind() == protoreflect.FloatKind {
		return protoreflect.ValueOfFloat32(float32(f))
	}
	return protoreflect.ValueOfFloat64(f)
}

// decodeTime writes a TIME value into a google.protobuf.Timestamp field.
func decodeTime(m protoreflect.Message, fd protoreflect.FieldDescriptor, val *core.Value) error {
	t, err := val.AsTime()
	if err != nil {
		return err
	}
	sub := m.Mutable(fd).Message()
	fields := sub.Descriptor().Fields()
	sub.Set(fields.ByName("seconds"), protoreflect.ValueOfInt64(t.Unix()))
	sub.Set(fields.ByName("nanos"), protoreflect.ValueOfInt32(int32(t.Nanosecond())))
	return nil
}

// decodeEnum matches a STRING value name back to an enum number.
func decodeEnum(m protoreflect.Message, fd protoreflect.FieldDescriptor, val *core.Value) error {
	name, err := val.AsStr()
	if err != nil {
		return err
	}
	vd := fd.Enum().Values().ByName(protoreflect.Name(name))
	if vd == nil {
		return nil
	}
	m.Set(fd, protoreflect.ValueOfEnum(vd.Number()))
	return nil
}

// decodeJSON reconstructs a map or message field from a JSON leaf value.
func decodeJSON(m protoreflect.Message, fp fieldPlan, val *core.Value) error {
	raw, err := val.AsJSON()
	if err != nil {
		return err
	}

	if fp.fd.IsMap() {
		return decodeMap(m, fp.fd, raw)
	}

	sub := m.Mutable(fp.fd).Message()
	return protojson.Unmarshal(raw, sub.Interface())
}

// decodeMap reconstructs a proto map field from a JSON object.
func decodeMap(m protoreflect.Message, fd protoreflect.FieldDescriptor, raw json.RawMessage) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}

	mp := m.Mutable(fd).Map()
	keyFd, valFd := fd.MapKey(), fd.MapValue()

	for k, vr := range obj {
		key, err := mapKey(keyFd, k)
		if err != nil {
			return err
		}
		val, err := mapValue(mp, valFd, vr)
		if err != nil {
			return err
		}
		mp.Set(key, val)
	}

	return nil
}

// mapKey converts a JSON object key string into a proto map key. String keys
// arrive unquoted; numeric keys arrive as bare number literals.
func mapKey(keyFd protoreflect.FieldDescriptor, k string) (protoreflect.MapKey, error) {
	_, cat, _ := scalarType(keyFd.Kind())

	cv := core.StringVal(k)
	if keyFd.Kind() != protoreflect.StringKind {
		var err error
		if cv, err = scalarMemberValue(cat, json.RawMessage(k)); err != nil {
			return protoreflect.MapKey{}, err
		}
	}

	pv, err := scalarProtoValue(keyFd, cat, cv)
	if err != nil {
		return protoreflect.MapKey{}, err
	}
	return pv.MapKey(), nil
}

// mapValue converts a JSON map value into a proto map value.
func mapValue(mp protoreflect.Map, valFd protoreflect.FieldDescriptor, raw json.RawMessage) (protoreflect.Value, error) {
	if valFd.Kind() == protoreflect.MessageKind || valFd.Kind() == protoreflect.GroupKind {
		mv := mp.NewValue()
		if err := protojson.Unmarshal(raw, mv.Message().Interface()); err != nil {
			return protoreflect.Value{}, err
		}
		return mv, nil
	}

	_, cat, _ := scalarType(valFd.Kind())
	cv, err := scalarMemberValue(cat, raw)
	if err != nil {
		return protoreflect.Value{}, err
	}
	return scalarProtoValue(valFd, cat, cv)
}

// decodeScalarArray fills a repeated scalar/enum field from an ARRAY value.
func decodeScalarArray(m protoreflect.Message, fp fieldPlan, val *core.Value) error {
	elems, err := val.AsArray()
	if err != nil {
		return err
	}

	list := m.Mutable(fp.fd).List()
	for _, ev := range elems {
		if ev == nil {
			continue
		}
		if fp.cat == catEnum {
			name, err := ev.AsStr()
			if err != nil {
				return err
			}
			vd := fp.fd.Enum().Values().ByName(protoreflect.Name(name))
			if vd == nil {
				continue
			}
			list.Append(protoreflect.ValueOfEnum(vd.Number()))
			continue
		}
		pv, err := scalarProtoValue(fp.fd, fp.cat, ev)
		if err != nil {
			return err
		}
		list.Append(pv)
	}

	return nil
}

// decodeObjectArray fills a repeated message field from an ARRAY<JSON> value.
func decodeObjectArray(m protoreflect.Message, fp fieldPlan, val *core.Value) error {
	elems, err := val.AsArray()
	if err != nil {
		return err
	}

	list := m.Mutable(fp.fd).List()
	for _, ev := range elems {
		if ev == nil {
			continue
		}
		raw, err := ev.AsJSON()
		if err != nil {
			return err
		}
		elem := list.NewElement()
		if err := decodeElement(fp.items, raw, elem.Message()); err != nil {
			return err
		}
		list.Append(elem)
	}

	return nil
}

// decodeElement decodes one typed-JSON object-array element into m, mirroring
// encodeElement.
func decodeElement(items []fieldPlan, raw json.RawMessage, m protoreflect.Message) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}

	for _, fp := range items {
		if fp.kind == kindNested {
			if err := decodeElemNested(fp, obj, m); err != nil {
				return err
			}
			continue
		}

		mr, ok := obj[fp.attr]
		if !ok || isJSONNull(mr) {
			continue
		}
		cv, err := memberToCoreValue(fp, mr)
		if err != nil {
			return err
		}
		if err := decodeField(m, fp, cv); err != nil {
			return err
		}
	}

	return nil
}

// decodeElemNested rebuilds a nested message member from dotted-prefixed keys.
func decodeElemNested(fp fieldPlan, obj map[string]json.RawMessage, m protoreflect.Message) error {
	sub := make(map[string]json.RawMessage)
	p := fp.attr + "."
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

	return decodeElement(fp.items, subRaw, m.Mutable(fp.fd).Message())
}

// memberToCoreValue converts an object-array element member into a typed
// [core.Value] so decodeField can assign it.
func memberToCoreValue(fp fieldPlan, raw json.RawMessage) (*core.Value, error) {
	switch fp.kind {
	case kindScalarArray:
		var rawElems []json.RawMessage
		if err := json.Unmarshal(raw, &rawElems); err != nil {
			return nil, err
		}
		tid := fp.coreType.ElemTypeID()
		elems := make([]*core.Value, 0, len(rawElems))
		for _, re := range rawElems {
			ev, err := scalarMemberValue(fp.cat, re)
			if err != nil {
				return nil, err
			}
			elems = append(elems, ev)
		}
		return core.ArrayVal(tid, elems...), nil

	case kindObjectArray:
		var rawElems []json.RawMessage
		if err := json.Unmarshal(raw, &rawElems); err != nil {
			return nil, err
		}
		elems := make([]*core.Value, 0, len(rawElems))
		for _, re := range rawElems {
			elems = append(elems, core.JSONVal(re))
		}
		return core.ArrayVal(core.TIDJSON, elems...), nil

	default:
		return scalarMemberValue(fp.cat, raw)
	}
}

// scalarMemberValue converts a single JSON member into a typed [core.Value].
func scalarMemberValue(cat leafCat, raw json.RawMessage) (*core.Value, error) {
	switch cat {
	case catBool, catInt, catUint, catFloat:
		return numberMemberValue(cat, raw)
	case catJSON:
		return core.JSONVal(raw), nil
	default: // catBytes, catTime, catString, catEnum
		return stringMemberValue(cat, raw)
	}
}

// numberMemberValue decodes a boolean or numeric JSON member.
func numberMemberValue(cat leafCat, raw json.RawMessage) (*core.Value, error) {
	switch cat {
	case catBool:
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, err
		}
		return core.BoolVal(b), nil
	case catInt:
		var i int64
		if err := json.Unmarshal(raw, &i); err != nil {
			return nil, err
		}
		return core.IntVal(i), nil
	case catUint:
		var u uint64
		if err := json.Unmarshal(raw, &u); err != nil {
			return nil, err
		}
		return core.UintVal(u), nil
	default: // catFloat
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, err
		}
		return core.FloatVal(f), nil
	}
}

// stringMemberValue decodes a string-shaped JSON member (string, enum name,
// base64 bytes, or RFC3339 time).
func stringMemberValue(cat leafCat, raw json.RawMessage) (*core.Value, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}

	switch cat {
	case catBytes:
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, err
		}
		return core.BytesVal(b), nil
	case catTime:
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, err
		}
		return core.TimeVal(ts.UTC()), nil
	default: // catString, catEnum
		return core.StringVal(s), nil
	}
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
