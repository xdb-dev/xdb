package jsonschemaimport

import "encoding/json"

// node is the subset of a JSON Schema (draft 2020-12) that the importer
// understands. Fields outside this subset are ignored, except the ones that map
// to explicit errors (anyOf, oneOf, cross-document/cyclic $ref).
type node struct {
	ID          string                     `json:"$id"`
	Ref         string                     `json:"$ref"`
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	Type        typeSet                    `json:"type"`
	Format      string                     `json:"format"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
	Items       json.RawMessage            `json:"items"`
	Defs        map[string]json.RawMessage `json:"$defs"`
	AllOf       []json.RawMessage          `json:"allOf"`
	AnyOf       []json.RawMessage          `json:"anyOf"`
	OneOf       []json.RawMessage          `json:"oneOf"`

	AdditionalProperties json.RawMessage `json:"additionalProperties"`

	// Constraints below are recorded as annotations only; XDB does not enforce
	// them.
	Enum             []json.RawMessage `json:"enum"`
	Const            json.RawMessage   `json:"const"`
	Pattern          string            `json:"pattern"`
	Minimum          json.RawMessage   `json:"minimum"`
	Maximum          json.RawMessage   `json:"maximum"`
	ExclusiveMinimum json.RawMessage   `json:"exclusiveMinimum"`
	ExclusiveMaximum json.RawMessage   `json:"exclusiveMaximum"`
	MinLength        json.RawMessage   `json:"minLength"`
	MaxLength        json.RawMessage   `json:"maxLength"`
}

// typeSet holds the "type" keyword, which is either a single string or an array
// of strings.
type typeSet []string

// UnmarshalJSON accepts a string or an array of strings.
func (ts *typeSet) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*ts = typeSet{s}
		return nil
	}
	var a []string
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*ts = a
	return nil
}

// nonNull returns the declared types with "null" removed. A single non-null
// type is the common nullable pattern (e.g. ["string","null"]).
func (ts typeSet) nonNull() []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		if t != "null" {
			out = append(out, t)
		}
	}
	return out
}

func parseNode(raw json.RawMessage) (*node, error) {
	var n node
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, ErrInvalidJSON
	}
	return &n, nil
}

// isUnion reports whether the node uses anyOf or oneOf.
func (n *node) isUnion() bool {
	return len(n.AnyOf) > 0 || len(n.OneOf) > 0
}

// isObject reports whether the node describes an object: an explicit
// type:object, or the presence of properties or an allOf composition.
func (n *node) isObject() bool {
	for _, t := range n.Type.nonNull() {
		if t == "object" {
			return true
		}
	}
	return len(n.Properties) > 0 || len(n.AllOf) > 0
}

// isArray reports whether the node describes an array.
func (n *node) isArray() bool {
	for _, t := range n.Type.nonNull() {
		if t == "array" {
			return true
		}
	}
	return len(n.Items) > 0
}

// typedAdditional reports whether additionalProperties is a schema (a map-like
// object) rather than a boolean. Such a node imports as an opaque JSON field.
func (n *node) typedAdditional() bool {
	if len(n.AdditionalProperties) == 0 {
		return false
	}
	var b bool
	return json.Unmarshal(n.AdditionalProperties, &b) != nil
}
