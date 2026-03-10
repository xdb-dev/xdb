package xdbjson

import "github.com/xdb-dev/xdb/schema"

// Options configures the Encoder and Decoder behavior.
type Options struct {
	Def           *schema.Def
	NS            string
	Schema        string
	IDField       string
	NSField       string
	SchemaField   string
	IncludeNS     bool
	IncludeSchema bool
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
