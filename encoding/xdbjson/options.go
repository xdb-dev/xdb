package xdbjson

import "github.com/xdb-dev/xdb/schema"

// Options configures the Encoder and Decoder behavior.
type Options struct {
	// NS is the default namespace for records.
	// Used when JSON does not contain namespace metadata.
	NS string

	// Schema is the default schema for records.
	// Used when JSON does not contain schema metadata.
	Schema string

	// IDField is the JSON field name for the record ID.
	// Defaults to "_id" if not specified.
	IDField string

	// NSField is the JSON field name for the namespace.
	// Defaults to "_ns" if not specified.
	NSField string

	// SchemaField is the JSON field name for the schema.
	// Defaults to "_schema" if not specified.
	SchemaField string

	// IncludeNS controls whether the encoder includes the namespace in JSON output.
	IncludeNS bool

	// IncludeSchema controls whether the encoder includes the schema in JSON output.
	IncludeSchema bool

	// Def is the schema definition used for type-aware decoding.
	// When provided, JSON values are converted according to field types.
	// When nil, default JSON type inference is used.
	Def *schema.Def
}

// DefaultOptions returns the default options.
func DefaultOptions() Options {
	return Options{
		IDField:     "_id",
		NSField:     "_ns",
		SchemaField: "_schema",
	}
}

func (o Options) withDefaults() Options {
	if o.IDField == "" {
		o.IDField = "_id"
	}
	if o.NSField == "" {
		o.NSField = "_ns"
	}
	if o.SchemaField == "" {
		o.SchemaField = "_schema"
	}
	return o
}
