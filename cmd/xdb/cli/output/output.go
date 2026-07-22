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
	// FormatPage writes a paginated list result. Structured formats
	// (json, yaml) render an {items, total, next_offset} envelope;
	// streaming and human formats (ndjson, table) render the items.
	FormatPage(w io.Writer, p Page) error
	// FormatError writes an error.
	FormatError(w io.Writer, err error) error
}

// Page is a paginated list result.
type Page struct {
	Items      []any
	Total      int
	NextOffset int
}

// pageDoc is the wire shape structured formats render for a Page.
type pageDoc struct {
	Items      []any `json:"items" yaml:"items"`
	Total      int   `json:"total" yaml:"total"`
	NextOffset int   `json:"next_offset,omitempty" yaml:"next_offset,omitempty"`
}

// doc returns the marshal-ready form of the page.
func (p Page) doc() pageDoc {
	items := p.Items
	if items == nil {
		items = []any{}
	}

	return pageDoc{Items: items, Total: p.Total, NextOffset: p.NextOffset}
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
