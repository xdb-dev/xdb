package protoimport

import "google.golang.org/protobuf/reflect/protoreflect"

// Options configures an import. Build it from [Option] values.
type Options struct {
	allowJSON map[protoreflect.FullName]bool
	ns        string
}

// Option customizes an import.
type Option func(*Options)

// WithNamespace overrides the namespace that messages import into. By default
// the proto package (e.g. "com.example") becomes the namespace.
func WithNamespace(ns string) Option {
	return func(o *Options) { o.ns = ns }
}

// WithAllowJSON marks fully-qualified message names that should import as an
// opaque JSON field rather than being walked into schema fields. It is the
// opt-in that breaks a recursive-message cycle.
func WithAllowJSON(messages ...string) Option {
	return func(o *Options) {
		if o.allowJSON == nil {
			o.allowJSON = make(map[protoreflect.FullName]bool)
		}
		for _, m := range messages {
			o.allowJSON[protoreflect.FullName(m)] = true
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
