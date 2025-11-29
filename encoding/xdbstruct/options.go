package xdbstruct

// Options configures the Encoder and Decoder behavior.
type Options struct {
	// Tag is the struct tag name to use for field configuration.
	// Defaults to "xdb" if not specified.
	Tag string

	// NS is the default namespace for records.
	// Can be overridden by structs implementing NSGetter.
	NS string

	// Schema is the default schema for records.
	// Can be overridden by structs implementing SchemaGetter.
	Schema string

	// Registry is the well-known type registry for custom type handling.
	// If nil, all nested structs will be flattened.
	// TODO: Add when encoding/wkt package is implemented.
	// Registry *wkt.Registry
}

// DefaultOptions returns the default options with Tag set to "xdb".
func DefaultOptions() Options {
	return Options{
		Tag: "xdb",
	}
}
