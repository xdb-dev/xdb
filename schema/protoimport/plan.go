package protoimport

import (
	"strconv"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/xdb-dev/xdb/core"
)

// planKind classifies how a proto field maps onto the schema and the wire.
type planKind int

const (
	// kindLeaf is a single [core.Value]: scalar, enum, time, bytes, or JSON.
	kindLeaf planKind = iota
	// kindNested is a message that flattens to dotted attributes.
	kindNested
	// kindScalarArray is a repeated scalar/enum/time/bytes → ARRAY<scalar>.
	kindScalarArray
	// kindObjectArray is a repeated message → ARRAY<JSON> with element Items.
	kindObjectArray
)

// leafCat is the concrete category of a leaf value, driving marshal/unmarshal.
type leafCat int

const (
	catBool leafCat = iota
	catInt
	catUint
	catFloat
	catString
	catBytes
	catTime // google.protobuf.Timestamp → TIME
	catEnum // enum → STRING (value name)
	catJSON // map, Struct, Any, Duration, allow-json message → JSON
)

// fieldPlan is the resolved plan for one proto field, shared by the schema
// walk, Marshal, and Unmarshal so the three can never diverge.
type fieldPlan struct {
	fd       protoreflect.FieldDescriptor
	attr     string
	coreType core.Type
	ann      map[string]string
	items    []fieldPlan // kindNested children / kindObjectArray element fields
	kind     planKind
	cat      leafCat // leaf category, or element category for kindScalarArray
	wrapped  bool    // leaf reads/writes the inner "value" of a wrapper message
}

// buildMessagePlan resolves the field plans for every field of md. stack holds
// the message full names being expanded (for cycle detection), including md.
func buildMessagePlan(
	md protoreflect.MessageDescriptor,
	stack []protoreflect.FullName,
	allow map[protoreflect.FullName]bool,
) ([]fieldPlan, error) {
	fields := md.Fields()
	out := make([]fieldPlan, 0, fields.Len())

	for i := range fields.Len() {
		fd := fields.Get(i)

		if oo := fd.ContainingOneof(); oo != nil && !oo.IsSynthetic() {
			return nil, errors.Wrap(ErrOneof,
				"field", string(fd.Name()),
				"oneof", string(oo.Name()),
				"fix", "flatten the oneof into separate optional fields",
			)
		}

		fp, err := buildFieldPlan(fd, stack, allow)
		if err != nil {
			return nil, err
		}
		out = append(out, fp)
	}

	return out, nil
}

// buildFieldPlan resolves a single field, dispatching on map / list / singular.
func buildFieldPlan(
	fd protoreflect.FieldDescriptor,
	stack []protoreflect.FullName,
	allow map[protoreflect.FullName]bool,
) (fieldPlan, error) {
	fp := fieldPlan{
		fd:   fd,
		attr: string(fd.Name()),
		ann:  map[string]string{"proto.number": strconv.Itoa(int(fd.Number()))},
	}

	switch {
	case fd.IsMap():
		fp.kind = kindLeaf
		fp.cat = catJSON
		fp.coreType = core.TypeJSON
		fp.ann["proto.type"] = "map"
		fp.ann["proto.map"] = mapSignature(fd)
		return fp, nil

	case fd.IsList():
		return buildListPlan(fp, fd, stack, allow)

	default:
		return buildSingularPlan(fp, fd, stack, allow)
	}
}

// buildListPlan resolves a repeated field into a scalar or object array.
func buildListPlan(
	fp fieldPlan,
	fd protoreflect.FieldDescriptor,
	stack []protoreflect.FullName,
	allow map[protoreflect.FullName]bool,
) (fieldPlan, error) {
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		md := fd.Message()

		// A repeated well-known/allow-json message whose element maps to a leaf
		// becomes an ARRAY of that leaf; a repeated regular message is an object
		// array.
		if cat, ct, ann, ok := leafMessage(md, allow); ok {
			fp.kind = kindScalarArray
			fp.cat = cat
			fp.coreType = core.NewArrayType(ct.ID())
			mergeAnn(fp.ann, ann)
			return fp, nil
		}

		items, err := expandMessage(md, string(fd.Name()), stack, allow)
		if err != nil {
			return fp, err
		}
		fp.kind = kindObjectArray
		fp.coreType = core.NewArrayType(core.TIDJSON)
		fp.items = items
		fp.ann["proto.type"] = string(md.FullName())
		return fp, nil
	}

	if fd.Kind() == protoreflect.EnumKind {
		fp.kind = kindScalarArray
		fp.cat = catEnum
		fp.coreType = core.NewArrayType(core.TIDString)
		fp.ann["proto.type"] = "enum"
		fp.ann["proto.enum"] = string(fd.Enum().FullName())
		return fp, nil
	}

	ct, cat, name := scalarType(fd.Kind())
	fp.kind = kindScalarArray
	fp.cat = cat
	fp.coreType = core.NewArrayType(ct.ID())
	fp.ann["proto.type"] = name
	return fp, nil
}

// buildSingularPlan resolves a non-repeated, non-map field.
func buildSingularPlan(
	fp fieldPlan,
	fd protoreflect.FieldDescriptor,
	stack []protoreflect.FullName,
	allow map[protoreflect.FullName]bool,
) (fieldPlan, error) {
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		md := fd.Message()

		if cat, ct, ann, ok := leafMessage(md, allow); ok {
			fp.kind = kindLeaf
			fp.cat = cat
			fp.coreType = ct
			fp.wrapped = isWrapper(md.FullName())
			mergeAnn(fp.ann, ann)
			return fp, nil
		}

		items, err := expandMessage(md, string(fd.Name()), stack, allow)
		if err != nil {
			return fp, err
		}
		fp.kind = kindNested
		fp.items = items
		return fp, nil
	}

	if fd.Kind() == protoreflect.EnumKind {
		fp.kind = kindLeaf
		fp.cat = catEnum
		fp.coreType = core.TypeString
		fp.ann["proto.type"] = "enum"
		fp.ann["proto.enum"] = string(fd.Enum().FullName())
		return fp, nil
	}

	ct, cat, name := scalarType(fd.Kind())
	fp.kind = kindLeaf
	fp.cat = cat
	fp.coreType = ct
	fp.ann["proto.type"] = name
	return fp, nil
}

// expandMessage walks md into its field plans, guarding against cycles.
func expandMessage(
	md protoreflect.MessageDescriptor,
	field string,
	stack []protoreflect.FullName,
	allow map[protoreflect.FullName]bool,
) ([]fieldPlan, error) {
	name := md.FullName()
	if contains(stack, name) {
		return nil, errors.Wrap(ErrRecursive,
			"field", field,
			"cycle", cycleString(stack, name),
			"fix", "pass WithAllowJSON(\""+string(name)+"\") to store it as JSON",
		)
	}
	return buildMessagePlan(md, append(stack, name), allow)
}

// leafMessage reports whether a message type maps to a leaf value rather than
// being walked: a well-known type, or an allow-json message. It returns the
// leaf category, its core type, and annotations to merge.
func leafMessage(
	md protoreflect.MessageDescriptor,
	allow map[protoreflect.FullName]bool,
) (leafCat, core.Type, map[string]string, bool) {
	name := md.FullName()

	if allow[name] {
		return catJSON, core.TypeJSON, map[string]string{
			"proto.type": string(name),
			"proto.json": "true",
		}, true
	}

	switch name {
	case "google.protobuf.Timestamp":
		return catTime, core.TypeTime, map[string]string{"proto.type": string(name)}, true
	case "google.protobuf.Duration",
		"google.protobuf.Struct",
		"google.protobuf.Value",
		"google.protobuf.ListValue",
		"google.protobuf.Any":
		return catJSON, core.TypeJSON, map[string]string{"proto.type": string(name)}, true
	}

	if cat, ct, ok := wrapperType(name); ok {
		return cat, ct, map[string]string{"proto.type": string(name)}, true
	}

	return 0, core.Type{}, nil, false
}

// isWrapper reports whether name is a google.protobuf wrapper message.
func isWrapper(name protoreflect.FullName) bool {
	_, _, ok := wrapperType(name)
	return ok
}

// wrapperType maps a wrapper message name to the leaf category and core type of
// its inner "value" field.
func wrapperType(name protoreflect.FullName) (leafCat, core.Type, bool) {
	switch name {
	case "google.protobuf.DoubleValue", "google.protobuf.FloatValue":
		return catFloat, core.TypeFloat, true
	case "google.protobuf.Int64Value", "google.protobuf.Int32Value":
		return catInt, core.TypeInt, true
	case "google.protobuf.UInt64Value", "google.protobuf.UInt32Value":
		return catUint, core.TypeUnsigned, true
	case "google.protobuf.BoolValue":
		return catBool, core.TypeBool, true
	case "google.protobuf.StringValue":
		return catString, core.TypeString, true
	case "google.protobuf.BytesValue":
		return catBytes, core.TypeBytes, true
	default:
		return 0, core.Type{}, false
	}
}

// scalarType maps a proto scalar kind to its core type, leaf category, and the
// proto type name recorded in annotations.
func scalarType(k protoreflect.Kind) (core.Type, leafCat, string) {
	switch k {
	case protoreflect.BoolKind:
		return core.TypeBool, catBool, "bool"
	case protoreflect.Int32Kind:
		return core.TypeInt, catInt, "int32"
	case protoreflect.Sint32Kind:
		return core.TypeInt, catInt, "sint32"
	case protoreflect.Sfixed32Kind:
		return core.TypeInt, catInt, "sfixed32"
	case protoreflect.Int64Kind:
		return core.TypeInt, catInt, "int64"
	case protoreflect.Sint64Kind:
		return core.TypeInt, catInt, "sint64"
	case protoreflect.Sfixed64Kind:
		return core.TypeInt, catInt, "sfixed64"
	case protoreflect.Uint32Kind:
		return core.TypeUnsigned, catUint, "uint32"
	case protoreflect.Fixed32Kind:
		return core.TypeUnsigned, catUint, "fixed32"
	case protoreflect.Uint64Kind:
		return core.TypeUnsigned, catUint, "uint64"
	case protoreflect.Fixed64Kind:
		return core.TypeUnsigned, catUint, "fixed64"
	case protoreflect.FloatKind:
		return core.TypeFloat, catFloat, "float"
	case protoreflect.DoubleKind:
		return core.TypeFloat, catFloat, "double"
	case protoreflect.StringKind:
		return core.TypeString, catString, "string"
	case protoreflect.BytesKind:
		return core.TypeBytes, catBytes, "bytes"
	default:
		return core.TypeString, catString, k.String()
	}
}

// mapSignature renders a map field's key/value types as "map<K, V>".
func mapSignature(fd protoreflect.FieldDescriptor) string {
	key := fd.MapKey().Kind().String()
	val := fd.MapValue().Kind().String()
	if fd.MapValue().Kind() == protoreflect.MessageKind {
		val = string(fd.MapValue().Message().FullName())
	}
	return "map<" + key + ", " + val + ">"
}

// mergeAnn copies src into dst.
func mergeAnn(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

// contains reports whether name is present in stack.
func contains(stack []protoreflect.FullName, name protoreflect.FullName) bool {
	for _, s := range stack {
		if s == name {
			return true
		}
	}
	return false
}

// cycleString renders the cycle from the first occurrence of name to name again.
func cycleString(stack []protoreflect.FullName, name protoreflect.FullName) string {
	parts := make([]string, 0, len(stack)+1)
	started := false
	for _, s := range stack {
		if s == name {
			started = true
		}
		if started {
			parts = append(parts, string(s))
		}
	}
	parts = append(parts, string(name))
	return strings.Join(parts, " → ")
}
