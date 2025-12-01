package xdbstruct

import (
	"github.com/xdb-dev/xdb/core"
)

// Decoder converts XDB records to Go structs.
type Decoder struct {
	opts Options
}

// NewDecoder creates a decoder with custom options.
func NewDecoder(opts Options) *Decoder {
	if opts.Tag == "" {
		opts.Tag = "xdb"
	}
	return &Decoder{opts: opts}
}

// NewDefaultDecoder creates a decoder with default options (Tag: "xdb").
func NewDefaultDecoder() *Decoder {
	return NewDecoder(DefaultOptions())
}

// FromRecord populates a struct from a core.Record.
// Returns an error if:
//   - v is not a pointer to struct
//   - Type mismatch between record value and struct field
//   - Custom unmarshaler fails
func (d *Decoder) FromRecord(record *core.Record, v any) error {
	// TODO: Implement full decoding logic
	// 1. Validate v is pointer to struct
	// 2. Iterate record tuples
	// 3. Map tuple attributes to struct fields
	// 4. Handle nested paths (unflatten dot notation)
	// 5. Handle type conversions using core.Value methods
	// 6. Handle arrays/slices
	// 7. Handle unmarshalers (json.Unmarshaler, encoding.BinaryUnmarshaler)
	// 8. Call setters (SetID, SetNS, SetSchema) after field population
	return ErrNotImplemented
}
