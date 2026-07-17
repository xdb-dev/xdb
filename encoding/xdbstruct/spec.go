package xdbstruct

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	timeType   = reflect.TypeFor[time.Time]()
	rawMsgType = reflect.TypeFor[json.RawMessage]()
)

// specKind classifies how a struct field maps onto the schema and the wire.
type specKind int

const (
	// kindLeaf is a single [core.Value]: scalar, time, bytes, or JSON.
	kindLeaf specKind = iota
	// kindNested is a struct that flattens to dotted attributes.
	kindNested
	// kindScalarArray is a slice of scalars → ARRAY<scalar>.
	kindScalarArray
	// kindObjectArray is a slice of structs → ARRAY<JSON> with element Items.
	kindObjectArray
)

// spec is the resolved plan for one struct field, shared by Def, Marshal, and
// Unmarshal so the three walks can never diverge.
type spec struct {
	goType     reflect.Type
	elemType   reflect.Type
	coreType   core.Type
	attr       string
	goTypeName string
	index      []int
	children   []spec
	kind       specKind
	ptr        bool
	required   bool
	elemPtr    bool
}

// tagOptions holds the parsed `xdb` struct tag.
type tagOptions struct {
	name     string
	skip     bool
	required bool
	jsonOpt  bool
}

// parseTag parses the `xdb` tag on a struct field. The grammar mirrors
// encoding/json: `xdb:"name,opt1,opt2"`. `xdb:"-"` skips the field. Recognized
// options are `required` and `json` (map/interface JSON opt-in). An empty name
// defaults to the Go field name verbatim (also the encoding/json convention).
func parseTag(f reflect.StructField) tagOptions {
	tag := f.Tag.Get("xdb")
	if tag == "-" {
		return tagOptions{skip: true}
	}

	parts := strings.Split(tag, ",")
	opts := tagOptions{name: parts[0]}
	for _, o := range parts[1:] {
		switch o {
		case "required":
			opts.required = true
		case "json":
			opts.jsonOpt = true
		}
	}
	return opts
}

// buildStruct resolves the specs for every exported field of t. stack contains
// every ancestor struct type INCLUDING t, so recursion into a type already on
// the stack is a cycle.
func buildStruct(t reflect.Type, stack []reflect.Type) ([]spec, error) {
	out := make([]spec, 0, t.NumField())

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		opts := parseTag(f)
		if opts.skip {
			continue
		}

		if f.Anonymous && opts.name == "" {
			promoted, err := buildEmbedded(f, i, stack)
			if err != nil {
				return nil, err
			}
			if promoted != nil {
				out = append(out, promoted...)
				continue
			}
		}

		name := opts.name
		if name == "" {
			name = f.Name
		}

		s, err := buildField(f.Type, name, opts, []int{i}, stack)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, nil
}

// buildEmbedded promotes the fields of an anonymous struct field per Go's field
// promotion rules. It returns nil (not an error) when f is not an embedded
// struct, so the caller falls back to treating it as a named field.
func buildEmbedded(f reflect.StructField, index int, stack []reflect.Type) ([]spec, error) {
	et := f.Type
	for et.Kind() == reflect.Pointer {
		et = et.Elem()
	}
	if et.Kind() != reflect.Struct || et == timeType {
		return nil, nil
	}

	if contains(stack, et) {
		return nil, cycleErr(stack, et, f.Name)
	}

	children, err := buildStruct(et, appendType(stack, et))
	if err != nil {
		return nil, err
	}

	for i := range children {
		children[i].index = append([]int{index}, children[i].index...)
	}
	return children, nil
}

// buildField resolves a single field. attr is its attribute name; stack holds
// the ancestor struct types (including the field's own struct) for cycle checks.
func buildField(
	ft reflect.Type,
	attr string,
	opts tagOptions,
	index []int,
	stack []reflect.Type,
) (spec, error) {
	s := spec{attr: attr, index: index, required: opts.required}

	for ft.Kind() == reflect.Pointer {
		s.ptr = true
		ft = ft.Elem()
	}
	s.goType = ft

	// The json opt-in short-circuits everything, including cycle detection: it
	// is the escape hatch for maps, interfaces, and recursive types.
	if opts.jsonOpt {
		s.kind = kindLeaf
		s.coreType = core.TypeJSON
		return s, nil
	}

	switch ft {
	case timeType:
		s.kind = kindLeaf
		s.coreType = core.TypeTime
		return s, nil
	case rawMsgType:
		s.kind = kindLeaf
		s.coreType = core.TypeJSON
		return s, nil
	}

	switch ft.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		s.kind = kindLeaf
		s.coreType = core.NewType(scalarKindTID(ft.Kind()))
		if isNamed(ft) {
			s.goTypeName = ft.String()
		}
		return s, nil

	case reflect.Slice, reflect.Array:
		return buildSlice(ft, s, stack)

	case reflect.Struct:
		if contains(stack, ft) {
			return s, cycleErr(stack, ft, attr)
		}
		children, err := buildStruct(ft, appendType(stack, ft))
		if err != nil {
			return s, err
		}
		s.kind = kindNested
		s.children = children
		return s, nil

	default:
		return s, unsupportedKind(attr, ft.Kind())
	}
}

// unsupportedKind builds the rejection error for a field kind that has no XDB
// mapping, naming the field and the fix.
func unsupportedKind(attr string, k reflect.Kind) error {
	switch k {
	case reflect.Map:
		return rejectErr(attr, "map",
			"add `,json` to the xdb tag to store it as JSON")
	case reflect.Interface:
		return rejectErr(attr, "interface",
			"add `,json` to the xdb tag to store it as JSON")
	case reflect.Chan:
		return rejectErr(attr, "channel",
			"remove the field or exclude it with `xdb:\"-\"`")
	case reflect.Func:
		return rejectErr(attr, "function",
			"remove the field or exclude it with `xdb:\"-\"`")
	default:
		return rejectErr(attr, k.String(), "use a supported type")
	}
}

// buildSlice resolves a slice field into bytes, a scalar array, or an object
// array.
func buildSlice(ft reflect.Type, s spec, stack []reflect.Type) (spec, error) {
	elem := ft.Elem()

	// []byte and named byte slices map to BYTES, not an array.
	if elem.Kind() == reflect.Uint8 {
		s.kind = kindLeaf
		s.coreType = core.TypeBytes
		return s, nil
	}

	for elem.Kind() == reflect.Pointer {
		s.elemPtr = true
		elem = elem.Elem()
	}
	s.elemType = elem

	if elem.Kind() == reflect.Struct && elem != timeType {
		if contains(stack, elem) {
			return s, cycleErr(stack, elem, s.attr)
		}
		children, err := buildStruct(elem, appendType(stack, elem))
		if err != nil {
			return s, err
		}
		s.kind = kindObjectArray
		s.coreType = core.NewArrayType(core.TIDJSON)
		s.children = children
		return s, nil
	}

	tid, ok := scalarElemTID(elem)
	if !ok {
		return s, rejectErr(s.attr, "[]"+elem.Kind().String(),
			"use a slice of a supported scalar or struct, or add `,json`")
	}
	s.kind = kindScalarArray
	s.coreType = core.NewArrayType(tid)
	return s, nil
}

// scalarKindTID maps a scalar reflect.Kind to its [core.TID].
func scalarKindTID(k reflect.Kind) core.TID {
	switch k {
	case reflect.Bool:
		return core.TIDBoolean
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return core.TIDInteger
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return core.TIDUnsigned
	case reflect.Float32, reflect.Float64:
		return core.TIDFloat
	case reflect.String:
		return core.TIDString
	default:
		return core.TIDUnknown
	}
}

// scalarElemTID maps a dereferenced slice element type to an array element
// [core.TID], covering time.Time, json.RawMessage, [][]byte, and scalars.
func scalarElemTID(elem reflect.Type) (core.TID, bool) {
	switch elem {
	case timeType:
		return core.TIDTime, true
	case rawMsgType:
		return core.TIDJSON, true
	}
	if elem.Kind() == reflect.Slice && elem.Elem().Kind() == reflect.Uint8 {
		return core.TIDBytes, true
	}
	tid := scalarKindTID(elem.Kind())
	if tid == core.TIDUnknown {
		return core.TIDUnknown, false
	}
	return tid, true
}

// isNamed reports whether ft is a named type distinct from its predeclared
// underlying scalar (e.g. `type UserID string`).
func isNamed(ft reflect.Type) bool {
	return ft.Name() != "" && ft.Name() != ft.Kind().String()
}

// contains reports whether t is present in stack.
func contains(stack []reflect.Type, t reflect.Type) bool {
	for _, s := range stack {
		if s == t {
			return true
		}
	}
	return false
}

// appendType returns a fresh stack with t appended, never aliasing stack.
func appendType(stack []reflect.Type, t reflect.Type) []reflect.Type {
	out := make([]reflect.Type, 0, len(stack)+1)
	out = append(out, stack...)
	return append(out, t)
}

// cycleErr builds an [ErrRecursive] naming the field and the type cycle.
func cycleErr(stack []reflect.Type, t reflect.Type, field string) error {
	names := make([]string, 0, len(stack)+1)
	started := false
	for _, s := range stack {
		if s == t {
			started = true
		}
		if started {
			names = append(names, s.Name())
		}
	}
	names = append(names, t.Name())

	return errors.Wrap(ErrRecursive,
		"field", field,
		"cycle", strings.Join(names, " → "),
		"fix", "add `xdb:\"...,json\"` to store the field as JSON",
	)
}
