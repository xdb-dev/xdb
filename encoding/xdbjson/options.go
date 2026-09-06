package xdbjson

import "github.com/xdb-dev/xdb/schema"

// options holds the resolved configuration for a single call.
type options struct {
	def           *schema.Def
	opaqueJSON    map[string]bool
	ns            string
	schema        string
	idField       string
	nsField       string
	schemaField   string
	prefix        string
	indent        string
	fields        []string
	includeNS     bool
	includeSchema bool
}

// Option configures a call to [ImportSchema], [Marshal], [MarshalInto], or
// [Unmarshal]. Each option documents which of them read it; an option a call
// does not read is ignored.
type Option func(*options)

func applyOptions(opts []Option) options {
	o := options{
		idField:     "_id",
		nsField:     "_ns",
		schemaField: "_schema",
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// WithDef sets the schema definition used for type-aware decoding.
//
// Read by [Marshal] and [MarshalInto]. Declared fields decode as their
// declared type; a value that cannot be typed is an error.
func WithDef(def *schema.Def) Option {
	return func(o *options) { o.def = def }
}

// WithNS sets the namespace.
//
// [ImportSchema] imports into this namespace and requires it. [Marshal] uses
// it as the default when the document has no namespace field.
func WithNS(ns string) Option {
	return func(o *options) { o.ns = ns }
}

// WithSchema sets the schema name.
//
// [ImportSchema] uses it to override the name otherwise derived from the
// document's title, then its $id filename. [Marshal] uses it as the default
// when the document has no schema field.
func WithSchema(s string) Option {
	return func(o *options) { o.schema = s }
}

// WithIDField sets the JSON field name used for the record ID. Defaults to
// "_id". Read by the data path.
func WithIDField(name string) Option {
	return func(o *options) { o.idField = name }
}

// WithNSField sets the JSON field name used for the namespace. Defaults to
// "_ns". Read by the data path.
func WithNSField(name string) Option {
	return func(o *options) { o.nsField = name }
}

// WithSchemaField sets the JSON field name used for the schema. Defaults to
// "_schema". Read by the data path.
func WithSchemaField(name string) Option {
	return func(o *options) { o.schemaField = name }
}

// WithIncludeNS includes the namespace in the encoded document. Read by
// [Unmarshal].
func WithIncludeNS() Option {
	return func(o *options) { o.includeNS = true }
}

// WithIncludeSchema includes the schema in the encoded document. Read by
// [Unmarshal].
func WithIncludeSchema() Option {
	return func(o *options) { o.includeSchema = true }
}

// WithIndent produces indented output. Read by [Unmarshal].
func WithIndent(prefix, indent string) Option {
	return func(o *options) {
		o.prefix = prefix
		o.indent = indent
	}
}

// WithFields limits the encoded document to the named fields. The ID field is
// always included. With no fields, every field is included. Read by
// [Unmarshal].
func WithFields(fields ...string) Option {
	return func(o *options) { o.fields = fields }
}

// WithOpaqueJSON marks same-document pointers (e.g. "#/$defs/Node") that
// [ImportSchema] should import as an opaque JSON field rather than walk into
// schema fields. It is the opt-in escape hatch that breaks a cyclic-$ref
// chain.
func WithOpaqueJSON(pointers ...string) Option {
	return func(o *options) {
		if o.opaqueJSON == nil {
			o.opaqueJSON = make(map[string]bool, len(pointers))
		}

		for _, p := range pointers {
			o.opaqueJSON[p] = true
		}
	}
}
