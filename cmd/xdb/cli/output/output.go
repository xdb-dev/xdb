// Package output provides formatters for CLI output in multiple formats.
package output

import "io"

// Format represents an output format.
type Format string

// Supported output formats.
const (
	FormatJSON   Format = "json"
	FormatNDJSON Format = "ndjson"
	FormatTable  Format = "table"
	FormatYAML   Format = "yaml"
)

// Formatter writes structured data in a specific format.
type Formatter interface {
	// FormatOne writes a single value.
	FormatOne(w io.Writer, v any) error
	// FormatList writes a list of values.
	FormatList(w io.Writer, items []any) error
	// FormatError writes an error.
	FormatError(w io.Writer, err error) error
}

// Detect returns the appropriate format based on the output flag
// and whether stdout is a TTY.
func Detect(flag string, isTTY bool) Format {
	if flag != "" {
		return Format(flag)
	}

	if isTTY {
		return FormatTable
	}

	return FormatJSON
}

// New creates a [Formatter] for the given format.
func New(f Format) Formatter {
	switch f {
	case FormatTable:
		return &tableFormatter{}
	case FormatYAML:
		return &yamlFormatter{}
	case FormatNDJSON:
		return &ndjsonFormatter{}
	default:
		return &jsonFormatter{}
	}
}
