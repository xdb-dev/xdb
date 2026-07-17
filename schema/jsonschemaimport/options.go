package jsonschemaimport

// Options configures an import. Build it from [Option] values.
type Options struct {
	allowJSON  map[string]bool
	ns         string
	schemaName string
}

// Option customizes an import.
type Option func(*Options)

// WithNamespace sets the namespace the schema imports into. It is required when
// the document's $id does not carry one (v1 never derives a namespace from
// $id).
func WithNamespace(ns string) Option {
	return func(o *Options) { o.ns = ns }
}

// WithSchemaName overrides the schema name. By default the name is derived from
// the document's title, then its $id filename.
func WithSchemaName(name string) Option {
	return func(o *Options) { o.schemaName = name }
}

// WithJSON marks same-document pointers (e.g. "#/$defs/Node") that should import
// as an opaque JSON field rather than being walked into schema fields. It is the
// opt-in escape hatch that breaks a cyclic-$ref chain.
func WithJSON(pointers ...string) Option {
	return func(o *Options) {
		if o.allowJSON == nil {
			o.allowJSON = make(map[string]bool)
		}
		for _, p := range pointers {
			o.allowJSON[p] = true
		}
	}
}

func buildOptions(opts []Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
