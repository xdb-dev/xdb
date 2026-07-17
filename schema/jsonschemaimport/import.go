package jsonschemaimport

import (
	"encoding/json"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// Import parses a JSON Schema document (a documented subset of draft 2020-12)
// into a [schema.Def]. The root must describe an object.
//
// The namespace comes from [WithNamespace]; the schema name from
// [WithSchemaName], the document title, or the $id filename. Every unsupported
// construct returns an error naming the JSON pointer to the offending node.
func Import(data []byte, opts ...Option) (*schema.Def, error) {
	o := buildOptions(opts)

	var rootAny any
	if err := json.Unmarshal(data, &rootAny); err != nil {
		return nil, ErrInvalidJSON
	}

	root, err := parseNode(data)
	if err != nil {
		return nil, err
	}

	im := &importer{root: rootAny, allowJSON: o.allowJSON}

	// A root $ref resolves once into the target node.
	rootPtr := "#"
	if root.Ref != "" {
		resolved, resPtr, asJSON, pop, derr := im.deref(data, rootPtr)
		if derr != nil {
			return nil, derr
		}
		pop()
		if asJSON {
			return nil, errors.Wrap(ErrUnsupported,
				"pointer", rootPtr,
				"reason", "root cannot import as JSON",
			)
		}
		root, rootPtr = resolved, resPtr
	}

	if !root.isObject() {
		return nil, errors.Wrap(ErrUnsupported,
			"pointer", rootPtr,
			"reason", "root schema must be an object",
		)
	}

	mode, err := modeFromAdditional(root, rootPtr)
	if err != nil {
		return nil, err
	}

	fields, err := im.objectFields(root, rootPtr)
	if err != nil {
		return nil, err
	}

	uri, err := resolveURI(o, root)
	if err != nil {
		return nil, err
	}

	def := &schema.Def{
		URI:         uri,
		Description: root.Description,
		Mode:        mode,
		Fields:      fields,
		Annotations: defAnnotations(root),
	}

	if err := def.Validate(); err != nil {
		return nil, err
	}

	return def, nil
}

// resolveURI builds the schema URI from the namespace and schema-name sources.
func resolveURI(o Options, root *node) (*core.URI, error) {
	ns, err := resolveNamespace(o)
	if err != nil {
		return nil, err
	}
	name, err := resolveSchemaName(o, root)
	if err != nil {
		return nil, err
	}
	return core.NewURI(ns, name)
}

// defAnnotations records the source marker and the $id when present.
func defAnnotations(root *node) map[string]string {
	ann := map[string]string{"source": "jsonschema"}
	if root.ID != "" {
		ann["jsonschema.id"] = root.ID
	}
	return ann
}

// importer carries the document root (for pointer resolution), the allow-JSON
// opt-in set, and the stack of $ref pointers currently being expanded (for
// cycle detection).
type importer struct {
	root      any
	allowJSON map[string]bool
	stack     []string
}

// built is the result of walking a single schema node: exactly one of field
// (a leaf/array/JSON field) or nested (a flattened object's field set) is set.
type built struct {
	field  *schema.Field
	nested map[string]schema.Field
}

// objectFields walks an object node into a flattened field map, applying its
// allOf merge, properties (nested objects flatten to dotted keys), and required
// list. It is the top-level (non-element) walk.
func (im *importer) objectFields(n *node, ptr string) (map[string]schema.Field, error) {
	out := make(map[string]schema.Field)

	for i, branch := range n.AllOf {
		bres, err := im.member(branch, ptr+"/allOf/"+strconv.Itoa(i), false)
		if err != nil {
			return nil, err
		}
		if bres.nested == nil {
			return nil, errors.Wrap(ErrUnsupported,
				"pointer", ptr+"/allOf/"+strconv.Itoa(i),
				"reason", "allOf branch must be an object",
			)
		}
		if err := merge(out, bres.nested); err != nil {
			return nil, err
		}
	}

	reqSet := make(map[string]struct{}, len(n.Required))
	for _, r := range n.Required {
		reqSet[r] = struct{}{}
	}

	single := make(map[string]struct{})
	var offending []string

	for _, name := range sortedKeys(n.Properties) {
		if !validKey(name) {
			offending = append(offending, name+" ("+ptr+"/properties/"+name+")")
			continue
		}

		res, err := im.member(n.Properties[name], ptr+"/properties/"+name, false)
		if err != nil {
			return nil, err
		}

		if res.nested != nil {
			_, objRequired := reqSet[name]
			prefixed := make(map[string]schema.Field, len(res.nested))
			for k, v := range res.nested {
				// A nested object's required members are enforced only when the
				// object itself is required; an optional nested object may be
				// omitted whole (its members become conditionally required,
				// which the flat IR cannot express, so it does not enforce them).
				if !objRequired {
					v.Required = false
				}
				prefixed[name+"."+k] = v
			}
			if err := merge(out, prefixed); err != nil {
				return nil, err
			}
			continue
		}

		if _, dup := out[name]; dup {
			return nil, errors.Wrap(ErrConflict, "field", name)
		}
		out[name] = *res.field
		single[name] = struct{}{}
	}

	if len(offending) > 0 {
		sort.Strings(offending)
		return nil, errors.Wrap(ErrInvalidKey,
			"pointer", ptr,
			"keys", strings.Join(offending, ", "),
		)
	}

	applyRequired(out, n.Required, single)

	return out, nil
}

// elementFields walks an object-array element schema into a field map. Unlike
// objectFields it does NOT flatten nested objects: the data path (encoding
// xdbjson) keeps element internals nested, so a nested object member imports as
// an opaque JSON field. Object-array members recurse.
func (im *importer) elementFields(n *node, ptr string) (map[string]schema.Field, error) {
	if len(n.AllOf) > 0 {
		return nil, errors.Wrap(ErrUnsupported,
			"pointer", ptr+"/allOf",
			"reason", "allOf is not supported inside array elements",
		)
	}

	out := make(map[string]schema.Field)
	present := make(map[string]struct{})
	var offending []string

	for _, name := range sortedKeys(n.Properties) {
		if !validKey(name) {
			offending = append(offending, name+" ("+ptr+"/properties/"+name+")")
			continue
		}

		res, err := im.member(n.Properties[name], ptr+"/properties/"+name, true)
		if err != nil {
			return nil, err
		}
		// inElement=true guarantees a single field (objects become JSON).
		out[name] = *res.field
		present[name] = struct{}{}
	}

	if len(offending) > 0 {
		sort.Strings(offending)
		return nil, errors.Wrap(ErrInvalidKey,
			"pointer", ptr,
			"keys", strings.Join(offending, ", "),
		)
	}

	applyRequired(out, n.Required, present)

	return out, nil
}

// member walks a single schema node. When inElement is true, object nodes
// import as opaque JSON fields (element internals are never flattened);
// otherwise object nodes flatten and return a nested field map.
func (im *importer) member(raw json.RawMessage, ptr string, inElement bool) (built, error) {
	n, resPtr, asJSON, pop, err := im.deref(raw, ptr)
	if err != nil {
		return built{}, err
	}
	defer pop()

	if asJSON {
		f := jsonField(map[string]string{"jsonschema.ref": n.Ref})
		return built{field: &f}, nil
	}

	if n.isUnion() {
		return built{}, unionError(resPtr, n)
	}

	switch {
	case n.isObject():
		if inElement || n.typedAdditional() {
			return built{field: objectJSONField(n)}, nil
		}
		m, err := im.objectFields(n, resPtr)
		if err != nil {
			return built{}, err
		}
		return built{nested: m}, nil

	case n.isArray():
		f, err := im.arrayField(n, resPtr)
		if err != nil {
			return built{}, err
		}
		return built{field: &f}, nil

	default:
		f, err := scalarField(n, resPtr)
		if err != nil {
			return built{}, err
		}
		return built{field: &f}, nil
	}
}

// arrayField builds the field for an array node from its items schema.
func (im *importer) arrayField(n *node, ptr string) (schema.Field, error) {
	if len(n.Items) == 0 {
		return schema.Field{}, errors.Wrap(ErrUnsupported,
			"pointer", ptr,
			"reason", "array requires a typed items schema",
		)
	}

	item, itemPtr, asJSON, pop, err := im.deref(n.Items, ptr+"/items")
	if err != nil {
		return schema.Field{}, err
	}
	defer pop()

	if asJSON {
		return arrayJSONField(), nil
	}
	if item.isUnion() {
		return schema.Field{}, unionError(itemPtr, item)
	}

	switch {
	case item.isObject():
		if item.typedAdditional() {
			return arrayJSONField(), nil
		}
		items, err := im.elementFields(item, itemPtr)
		if err != nil {
			return schema.Field{}, err
		}
		return schema.Field{
			Type:  core.NewArrayType(core.TIDJSON),
			Items: items,
		}, nil

	case item.isArray():
		return schema.Field{}, errors.Wrap(ErrUnsupported,
			"pointer", itemPtr,
			"reason", "arrays of arrays are not supported",
		)

	default:
		tid, err := scalarTID(item, itemPtr)
		if err != nil {
			return schema.Field{}, err
		}
		return schema.Field{Type: core.NewArrayType(tid)}, nil
	}
}

// deref follows a $ref chain to the underlying node. It returns a pop closure
// the caller must invoke (defer) to unwind any $ref pointers pushed onto the
// cycle-detection stack. asJSON is true when the chain hits a WithJSON pointer.
func (im *importer) deref(raw json.RawMessage, ptr string) (*node, string, bool, func(), error) {
	var pushed int
	pop := func() {
		im.stack = im.stack[:len(im.stack)-pushed]
	}

	cur, err := parseNode(raw)
	if err != nil {
		return nil, "", false, pop, err
	}
	curPtr := ptr

	for cur.Ref != "" {
		ref := cur.Ref

		if !strings.HasPrefix(ref, "#") {
			return nil, "", false, pop, errors.Wrap(ErrCrossDocument,
				"pointer", curPtr,
				"ref", ref,
			)
		}
		if im.allowJSON[ref] {
			return cur, ref, true, pop, nil
		}
		if im.onStack(ref) {
			return nil, "", false, pop, errors.Wrap(ErrCyclicRef,
				"pointer", curPtr,
				"ref", ref,
			)
		}

		target, err := im.resolvePointer(ref)
		if err != nil {
			return nil, "", false, pop, err
		}

		im.stack = append(im.stack, ref)
		pushed++

		cur, err = parseNode(target)
		if err != nil {
			return nil, "", false, pop, err
		}
		curPtr = ref
	}

	return cur, curPtr, false, pop, nil
}

func (im *importer) onStack(ref string) bool {
	for _, r := range im.stack {
		if r == ref {
			return true
		}
	}
	return false
}

// resolvePointer resolves a same-document JSON pointer (e.g. "#/$defs/Node")
// against the document root and returns the target node as raw JSON.
func (im *importer) resolvePointer(ref string) (json.RawMessage, error) {
	frag := strings.TrimPrefix(ref, "#")
	frag = strings.TrimPrefix(frag, "/")

	cur := im.root
	if frag != "" {
		for _, token := range strings.Split(frag, "/") {
			token = decodePointerToken(token)
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, errors.Wrap(ErrUnresolvedRef, "ref", ref)
			}
			next, ok := m[token]
			if !ok {
				return nil, errors.Wrap(ErrUnresolvedRef, "ref", ref)
			}
			cur = next
		}
	}

	return json.Marshal(cur)
}

// decodePointerToken applies RFC 6901 unescaping (~1 -> /, ~0 -> ~).
func decodePointerToken(t string) string {
	t = strings.ReplaceAll(t, "~1", "/")
	t = strings.ReplaceAll(t, "~0", "~")
	return t
}

// merge copies src into dst, rejecting any key already present.
func merge(dst, src map[string]schema.Field) error {
	for k, v := range src {
		if _, ok := dst[k]; ok {
			return errors.Wrap(ErrConflict, "field", k)
		}
		dst[k] = v
	}
	return nil
}

// applyRequired marks fields named in required as Required, but only when the
// name produced a single field (a flattened nested object cannot be marked
// required as a whole — its own members carry Required).
func applyRequired(fields map[string]schema.Field, required []string, single map[string]struct{}) {
	for _, name := range required {
		if _, ok := single[name]; !ok {
			continue
		}
		f := fields[name]
		f.Required = true
		fields[name] = f
	}
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// validKey reports whether name is a single attribute segment. A '.' is
// rejected because it is the path separator; anything core rejects as an
// attribute is rejected too. No escaping in v1.
func validKey(name string) bool {
	if name == "" || strings.Contains(name, ".") {
		return false
	}
	_, err := core.ParsePath("_/_/_#" + name)
	return err == nil
}

func unionError(ptr string, n *node) error {
	kind := "anyOf"
	if len(n.OneOf) > 0 {
		kind = "oneOf"
	}
	return errors.Wrap(ErrUnion, "pointer", ptr, "keyword", kind)
}

// modeFromAdditional maps the root additionalProperties keyword to a schema
// mode: false -> strict, true or absent -> flexible. A typed schema at the root
// has no record representation and is an error.
func modeFromAdditional(n *node, ptr string) (schema.Mode, error) {
	if len(n.AdditionalProperties) == 0 {
		return schema.ModeFlexible, nil
	}
	var b bool
	if err := json.Unmarshal(n.AdditionalProperties, &b); err == nil {
		if b {
			return schema.ModeFlexible, nil
		}
		return schema.ModeStrict, nil
	}
	return "", errors.Wrap(ErrUnsupported,
		"pointer", ptr+"/additionalProperties",
		"reason", "a typed additionalProperties schema at the root is not supported",
	)
}

func resolveNamespace(o Options) (string, error) {
	if o.ns != "" {
		return o.ns, nil
	}
	return "", errors.Wrap(ErrNoNamespace,
		"fix", "pass WithNamespace to set the target namespace",
	)
}

func resolveSchemaName(o Options, root *node) (string, error) {
	if o.schemaName != "" {
		return o.schemaName, nil
	}
	if n := sanitizeName(root.Title); n != "" {
		return n, nil
	}
	if n := nameFromID(root.ID); n != "" {
		return n, nil
	}
	return "", errors.Wrap(ErrNoSchemaName,
		"fix", "pass WithSchemaName, or add a title to the schema",
	)
}

// nameFromID derives a schema name from an $id URL's filename, stripping a
// trailing .json / .schema.json suffix.
func nameFromID(id string) string {
	if id == "" {
		return ""
	}
	u, err := url.Parse(id)
	if err != nil {
		return ""
	}
	base := path.Base(u.Path)
	base = strings.TrimSuffix(base, ".json")
	base = strings.TrimSuffix(base, ".schema")
	return sanitizeName(base)
}

// sanitizeName returns name unchanged if it is a valid URI schema component;
// otherwise "" so callers fall through to the next source.
func sanitizeName(name string) string {
	if name == "" {
		return ""
	}
	if _, err := core.NewURI("_", name); err != nil {
		return ""
	}
	return name
}
