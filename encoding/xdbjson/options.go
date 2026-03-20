package xdbjson

import "github.com/xdb-dev/xdb/schema"

// options holds internal configuration for [Encoder] and [Decoder].
type options struct {
	def           *schema.Def
	ns            string
	schema        string
	idField       string
	nsField       string
	schemaField   string
	includeNS     bool
	includeSchema bool
}

// Option is a functional option for configuring an [Encoder] or [Decoder].
type Option func(*options)

func defaultOptions() options {
	return options{
		idField:     "_id",
		nsField:     "_ns",
		schemaField: "_schema",
	}
}

func applyOptions(opts []Option) options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithDef sets the schema definition used for type-aware decoding.
func WithDef(def *schema.Def) Option {
	return func(o *options) { o.def = def }
}

// WithNS sets the default namespace for decoded records.
func WithNS(ns string) Option {
	return func(o *options) { o.ns = ns }
}

// WithSchema sets the default schema for decoded records.
func WithSchema(s string) Option {
	return func(o *options) { o.schema = s }
}

// WithIDField sets the JSON field name used for the record ID.
func WithIDField(name string) Option {
	return func(o *options) { o.idField = name }
}

// WithNSField sets the JSON field name used for the namespace.
func WithNSField(name string) Option {
	return func(o *options) { o.nsField = name }
}

// WithSchemaField sets the JSON field name used for the schema.
func WithSchemaField(name string) Option {
	return func(o *options) { o.schemaField = name }
}

// WithIncludeNS enables including the namespace in encoded JSON output.
func WithIncludeNS() Option {
	return func(o *options) { o.includeNS = true }
}

// WithIncludeSchema enables including the schema in encoded JSON output.
func WithIncludeSchema() Option {
	return func(o *options) { o.includeSchema = true }
}

// EncodeOption configures a single [Encoder.FromRecord] call.
type EncodeOption func(*encodeConfig)

type encodeConfig struct {
	prefix string
	indent string
	fields []string
}

// WithIndent enables indented JSON output with the given prefix and indent string.
func WithIndent(prefix, indent string) EncodeOption {
	return func(c *encodeConfig) {
		c.prefix = prefix
		c.indent = indent
	}
}

// WithFields limits encoded output to the specified fields.
// The ID field is always included. If no fields are specified, all fields are included.
func WithFields(fields ...string) EncodeOption {
	return func(c *encodeConfig) { c.fields = fields }
}
