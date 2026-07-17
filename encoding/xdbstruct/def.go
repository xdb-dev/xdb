package xdbstruct

import (
	"reflect"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// Def reflects over T and returns the [schema.Def] it projects to. T must be a
// struct (or pointer to one). The produced schema is [schema.ModeStrict]: every
// field a struct declares is validated on write.
//
// Mapping (see the package doc for the full grammar):
//   - exported fields only; `xdb:"name,required"` sets the attribute name and
//     Required; `xdb:"-"` skips; an empty name defaults to the Go field name.
//   - scalars → their TIDs; time.Time → TIME; []byte → BYTES;
//     json.RawMessage and `xdb:"...,json"` fields → JSON.
//   - nested structs (value or pointer) flatten to dotted attributes.
//   - []Struct → an object array (ARRAY<JSON> with element Items);
//     []scalar → ARRAY<scalar>.
//   - named scalar types record Annotations["go.type"].
//
// It returns an error for recursive types, and for maps/interfaces/channels/
// functions without the `json` opt-in, each naming the field and the fix.
func Def[T any](uri string) (*schema.Def, error) {
	u, err := core.ParseURI(uri)
	if err != nil {
		return nil, err
	}

	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, ErrNotStruct
	}

	specs, err := buildStruct(t, []reflect.Type{t})
	if err != nil {
		return nil, err
	}

	def := &schema.Def{
		URI:    u,
		Mode:   schema.ModeStrict,
		Fields: specsToFields(specs, ""),
	}

	if err := def.Validate(); err != nil {
		return nil, err
	}

	return def, nil
}

// specsToFields flattens specs into a schema field map. prefix carries the
// dotted namespace for nested structs; object-array Items form their own
// namespace (built with an empty prefix).
func specsToFields(specs []spec, prefix string) map[string]schema.Field {
	fields := make(map[string]schema.Field)

	for _, s := range specs {
		name := prefix + s.attr

		switch s.kind {
		case kindNested:
			for k, v := range specsToFields(s.children, name+".") {
				fields[k] = v
			}

		case kindObjectArray:
			fields[name] = schema.Field{
				Type:     s.coreType,
				Required: s.required,
				Items:    specsToFields(s.children, ""),
			}

		default: // kindLeaf, kindScalarArray
			f := schema.Field{
				Type:     s.coreType,
				Required: s.required,
			}
			if s.goTypeName != "" {
				f.Annotations = map[string]string{"go.type": s.goTypeName}
			}
			fields[name] = f
		}
	}

	return fields
}
