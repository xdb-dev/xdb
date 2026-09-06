package xdbjson

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"github.com/google/jsonschema-go/jsonschema"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// scalarField builds a leaf field for a scalar node, mapping its type (and
// format) to a core type and recording constraints as annotations.
func scalarField(n *jsonschema.Schema, ptr string) (schema.Field, error) {
	tid, err := scalarTID(n, ptr)
	if err != nil {
		return schema.Field{}, err
	}
	return schema.Field{
		Type:        core.NewType(tid),
		Annotations: annotations(n),
	}, nil
}

// scalarTID resolves a scalar node's core type. string+format:date-time maps to
// TIME. When type is absent it is inferred from enum/const values.
func scalarTID(n *jsonschema.Schema, ptr string) (core.TID, error) {
	types := nonNullTypes(n)

	if len(types) > 1 {
		return "", errors.Wrap(ErrUnsupported,
			"pointer", ptr,
			"reason", "multiple non-null types are not supported",
			"types", strings.Join(types, ","),
		)
	}

	var t string
	switch {
	case len(types) == 1:
		t = types[0]
	case len(n.Enum) > 0:
		t = inferJSONType(n.Enum[0])
	case n.Const != nil:
		t = inferJSONType(*n.Const)
	default:
		return "", errors.Wrap(ErrUnsupported,
			"pointer", ptr,
			"reason", "node has no mappable type",
		)
	}

	switch t {
	case "string":
		if n.Format == "date-time" {
			return core.TIDTime, nil
		}
		return core.TIDString, nil
	case "integer":
		return core.TIDInteger, nil
	case "number":
		return core.TIDFloat, nil
	case "boolean":
		return core.TIDBoolean, nil
	default:
		return "", errors.Wrap(ErrUnsupported,
			"pointer", ptr,
			"reason", "unsupported scalar type",
			"type", t,
		)
	}
}

// jsonField builds an opaque JSON field carrying extra annotations.
func jsonField(extra map[string]string) schema.Field {
	ann := map[string]string{"jsonschema.type": "json"}
	for k, v := range extra {
		if v != "" {
			ann[k] = v
		}
	}
	return schema.Field{
		Type:        core.NewType(core.TIDJSON),
		Annotations: ann,
	}
}

// objectJSONField builds the opaque JSON field for an object that imports as a
// map (typed additionalProperties) or as an element member.
func objectJSONField(n *jsonschema.Schema) *schema.Field {
	extra := map[string]string{}
	if typedAdditional(n) {
		extra["jsonschema.additionalProperties"] = "schema"
	}
	f := jsonField(extra)
	return &f
}

// arrayJSONField builds an ARRAY<JSON> field with no element schema, used for
// arrays of opaque items (maps, allow-JSON refs).
func arrayJSONField() schema.Field {
	return schema.Field{
		Type:        core.NewArrayType(core.TIDJSON),
		Annotations: map[string]string{"jsonschema.type": "json"},
	}
}

// annotations collects the constraint keywords XDB does not enforce into a
// fidelity-preserving annotation map. Returns nil when none are present.
func annotations(n *jsonschema.Schema) map[string]string {
	ann := map[string]string{}

	if n.Format != "" {
		ann["jsonschema.format"] = n.Format
	}
	if len(n.Enum) > 0 {
		if raw, err := json.Marshal(n.Enum); err == nil {
			ann["jsonschema.enum"] = string(raw)
		}
	}
	if n.Const != nil {
		if raw, err := json.Marshal(*n.Const); err == nil {
			ann["jsonschema.const"] = string(raw)
		}
	}
	if n.Pattern != "" {
		ann["jsonschema.pattern"] = n.Pattern
	}
	addFloat(ann, "jsonschema.minimum", n.Minimum)
	addFloat(ann, "jsonschema.maximum", n.Maximum)
	addFloat(ann, "jsonschema.exclusiveMinimum", n.ExclusiveMinimum)
	addFloat(ann, "jsonschema.exclusiveMaximum", n.ExclusiveMaximum)
	addInt(ann, "jsonschema.minLength", n.MinLength)
	addInt(ann, "jsonschema.maxLength", n.MaxLength)

	if len(ann) == 0 {
		return nil
	}
	return ann
}

func addFloat(ann map[string]string, key string, v *float64) {
	if v != nil {
		ann[key] = strconv.FormatFloat(*v, 'g', -1, 64)
	}
}

func addInt(ann map[string]string, key string, v *int) {
	if v != nil {
		ann[key] = strconv.Itoa(*v)
	}
}

// inferJSONType returns the JSON Schema type name of a decoded JSON value, used
// when enum/const are present without an explicit type.
func inferJSONType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	default:
		return ""
	}
}
