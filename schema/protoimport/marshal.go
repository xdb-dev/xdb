package protoimport

import (
	"encoding/json"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/xdb-dev/xdb/core"
)

// Marshal encodes a proto message into a [core.Record] addressed by uri. The
// uri must carry ns, schema, and id. If one of them is missing, Marshal
// panics. It walks the message through protoreflect with the same field plan
// that the importer produces, so the attributes of the record match the
// imported schema.
//
// Scalars, enums (as their value name), Timestamp (as TIME), and bytes map to
// leaf values; nested messages flatten to dotted attributes; repeated messages
// become an ARRAY<JSON> object array; map/Duration/Struct/Any/allow-json fields
// become JSON.
func Marshal(uri string, msg proto.Message, opts ...Option) (*core.Record, error) {
	u, err := core.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	o := buildOptions(opts)
	m := msg.ProtoReflect()
	md := m.Descriptor()

	plan, err := buildMessagePlan(md, []protoreflect.FullName{md.FullName()}, o.allowJSON)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(u.NS(), u.Schema(), u.ID())
	if err := encodeInto(record, plan, m, ""); err != nil {
		return nil, err
	}

	return record, nil
}

// encodeInto writes tuples for plan from m into record. prefix carries the
// dotted namespace for nested messages.
func encodeInto(record *core.Record, plan []fieldPlan, m protoreflect.Message, prefix string) error {
	for _, fp := range plan {
		if !m.Has(fp.fd) {
			continue // absent (unset, or proto3 implicit zero) → no tuple
		}

		switch fp.kind {
		case kindNested:
			sub := m.Get(fp.fd).Message()
			if err := encodeInto(record, fp.items, sub, prefix+fp.attr+"."); err != nil {
				return err
			}

		case kindScalarArray:
			v, err := scalarArrayValue(fp, m.Get(fp.fd).List())
			if err != nil {
				return err
			}
			record.Set(prefix+fp.attr, v)

		case kindObjectArray:
			v, err := objectArrayValue(fp, m.Get(fp.fd).List())
			if err != nil {
				return err
			}
			record.Set(prefix+fp.attr, v)

		default: // kindLeaf
			v, err := leafValue(fp, m.Get(fp.fd))
			if err != nil {
				return err
			}
			record.Set(prefix+fp.attr, v)
		}
	}

	return nil
}

// leafValue builds a leaf [core.Value] from a proto field value.
func leafValue(fp fieldPlan, val protoreflect.Value) (*core.Value, error) {
	if fp.wrapped {
		val = val.Message().Get(fp.fd.Message().Fields().ByName("value"))
	}

	switch fp.cat {
	case catBool:
		return core.BoolVal(val.Bool()), nil
	case catInt:
		return core.IntVal(val.Int()), nil
	case catUint:
		return core.UintVal(val.Uint()), nil
	case catFloat:
		return core.FloatVal(val.Float()), nil
	case catString:
		return core.StringVal(val.String()), nil
	case catBytes:
		return core.BytesVal(val.Bytes()), nil
	case catTime:
		return core.TimeVal(protoTime(val.Message())), nil
	case catEnum:
		return core.StringVal(enumName(fp.fd, val)), nil
	default: // catJSON
		raw, err := jsonLeaf(fp, val)
		if err != nil {
			return nil, err
		}
		return core.JSONVal(raw), nil
	}
}

// scalarArrayValue builds an ARRAY<scalar> value from a repeated scalar field.
func scalarArrayValue(fp fieldPlan, list protoreflect.List) (*core.Value, error) {
	tid := fp.coreType.ElemTypeID()
	elems := make([]*core.Value, 0, list.Len())

	for i := range list.Len() {
		v, err := leafValue(fp, list.Get(i))
		if err != nil {
			return nil, err
		}
		elems = append(elems, v)
	}

	return core.ArrayVal(tid, elems...), nil
}

// objectArrayValue builds an ARRAY<JSON> value from a repeated message field,
// encoding each element as a typed JSON object.
func objectArrayValue(fp fieldPlan, list protoreflect.List) (*core.Value, error) {
	elems := make([]*core.Value, 0, list.Len())

	for i := range list.Len() {
		obj, err := encodeElement(fp.items, list.Get(i).Message())
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		elems = append(elems, core.JSONVal(raw))
	}

	return core.ArrayVal(core.TIDJSON, elems...), nil
}

// encodeElement renders one object-array element as a map ready for
// json.Marshal. Leaf members become Go values that json.Marshal renders in
// the form the schema expects (Timestamp as RFC3339Nano, bytes as base64).
// Nested messages flatten to dotted keys. Nested arrays recurse.
func encodeElement(items []fieldPlan, m protoreflect.Message) (map[string]any, error) {
	out := make(map[string]any)

	for _, fp := range items {
		if !m.Has(fp.fd) {
			continue
		}

		switch fp.kind {
		case kindNested:
			sub, err := encodeElement(fp.items, m.Get(fp.fd).Message())
			if err != nil {
				return nil, err
			}
			for k, v := range sub {
				out[fp.attr+"."+k] = v
			}

		case kindObjectArray:
			list := m.Get(fp.fd).List()
			arr := make([]any, 0, list.Len())
			for i := range list.Len() {
				sub, err := encodeElement(fp.items, list.Get(i).Message())
				if err != nil {
					return nil, err
				}
				arr = append(arr, sub)
			}
			out[fp.attr] = arr

		case kindScalarArray:
			list := m.Get(fp.fd).List()
			arr := make([]any, 0, list.Len())
			for i := range list.Len() {
				gv, err := leafGo(fp, list.Get(i))
				if err != nil {
					return nil, err
				}
				arr = append(arr, gv)
			}
			out[fp.attr] = arr

		default: // kindLeaf
			gv, err := leafGo(fp, m.Get(fp.fd))
			if err != nil {
				return nil, err
			}
			out[fp.attr] = gv
		}
	}

	return out, nil
}

// leafGo returns a Go value for an object-array element member that json.Marshal
// renders in the form the schema expects for that member's type.
func leafGo(fp fieldPlan, val protoreflect.Value) (any, error) {
	if fp.wrapped {
		val = val.Message().Get(fp.fd.Message().Fields().ByName("value"))
	}

	switch fp.cat {
	case catBool:
		return val.Bool(), nil
	case catInt:
		return val.Int(), nil
	case catUint:
		return val.Uint(), nil
	case catFloat:
		return val.Float(), nil
	case catString:
		return val.String(), nil
	case catBytes:
		return val.Bytes(), nil
	case catTime:
		return protoTime(val.Message()).Format(time.RFC3339Nano), nil
	case catEnum:
		return enumName(fp.fd, val), nil
	default: // catJSON
		raw, err := jsonLeaf(fp, val)
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
}

// jsonLeaf renders a map/message JSON leaf into raw JSON.
func jsonLeaf(fp fieldPlan, val protoreflect.Value) (json.RawMessage, error) {
	if fp.fd.IsMap() {
		return marshalMap(fp.fd, val.Map())
	}
	return protojson.Marshal(val.Message().Interface())
}

// marshalMap encodes a proto map field as a JSON object.
func marshalMap(fd protoreflect.FieldDescriptor, m protoreflect.Map) (json.RawMessage, error) {
	valFd := fd.MapValue()
	out := make(map[string]any, m.Len())

	var rangeErr error
	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		gv, err := mapValueGo(valFd, v)
		if err != nil {
			rangeErr = err
			return false
		}
		out[k.String()] = gv
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}

	return json.Marshal(out)
}

// mapValueGo converts a proto map value into a json-encodable Go value.
func mapValueGo(valFd protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	if valFd.Kind() == protoreflect.MessageKind || valFd.Kind() == protoreflect.GroupKind {
		raw, err := protojson.Marshal(v.Message().Interface())
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return v.Interface(), nil
}

// protoTime converts a google.protobuf.Timestamp message into a UTC time.Time.
func protoTime(m protoreflect.Message) time.Time {
	fields := m.Descriptor().Fields()
	seconds := m.Get(fields.ByName("seconds")).Int()
	nanos := m.Get(fields.ByName("nanos")).Int()
	return time.Unix(seconds, nanos).UTC()
}

// enumName returns the value name for an enum field value.
func enumName(fd protoreflect.FieldDescriptor, val protoreflect.Value) string {
	vd := fd.Enum().Values().ByNumber(val.Enum())
	if vd == nil {
		return ""
	}
	return string(vd.Name())
}
