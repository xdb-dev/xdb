package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// Format represents an output format type.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatYAML  Format = "yaml"
)

// SelectDefaultFormat detects if stdout is a TTY and returns appropriate format.
func SelectDefaultFormat() Format {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return FormatTable
	}
	return FormatJSON
}

// Formatter formats data for output.
type Formatter interface {
	Format(data any) ([]byte, error)
	ContentType() string
}

// JSONFormatter outputs JSON with indentation.
type JSONFormatter struct {
	Indent string
}

func (f *JSONFormatter) Format(data any) ([]byte, error) {
	indent := f.Indent
	if indent == "" {
		indent = "  "
	}
	return json.MarshalIndent(data, "", indent)
}

func (f *JSONFormatter) ContentType() string {
	return "application/json"
}

// YAMLFormatter outputs YAML.
type YAMLFormatter struct{}

func (f *YAMLFormatter) Format(data any) ([]byte, error) {
	return yaml.Marshal(data)
}

func (f *YAMLFormatter) ContentType() string {
	return "application/yaml"
}

// TableFormatter renders data as human-readable tables.
type TableFormatter struct{}

func (f *TableFormatter) Format(data any) ([]byte, error) {
	switch v := data.(type) {
	case *core.Record:
		return f.formatRecord(v)
	case []*core.Record:
		return f.formatRecords(v)
	case *schema.Def:
		return f.formatSchemaDef(v)
	case []*schema.Def:
		return f.formatSchemaDefs(v)
	case map[string]any:
		return f.formatMap(v)
	default:
		// Fallback to JSON for unknown types
		return json.MarshalIndent(data, "", "  ")
	}
}

func (f *TableFormatter) formatRecord(r *core.Record) ([]byte, error) {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Attribute", "Value", "Type"})

	for _, tuple := range r.Tuples() {
		valueStr := fmt.Sprintf("%v", tuple.Value())
		typeStr := reflect.TypeOf(tuple.Value()).String()
		t.AppendRow(table.Row{
			tuple.Attr().String(),
			valueStr,
			typeStr,
		})
	}

	return []byte(t.Render()), nil
}

func (f *TableFormatter) formatRecords(records []*core.Record) ([]byte, error) {
	if len(records) == 0 {
		return []byte("No records found\n"), nil
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)

	// Use first record to determine columns
	firstRecord := records[0]
	headers := []any{"URI"}
	for _, tuple := range firstRecord.Tuples() {
		headers = append(headers, tuple.Attr().String())
	}
	t.AppendHeader(headers)

	// Add rows
	for _, record := range records {
		row := []any{record.URI().String()}
		for _, tuple := range record.Tuples() {
			row = append(row, fmt.Sprintf("%v", tuple.Value()))
		}
		t.AppendRow(row)
	}

	return []byte(t.Render()), nil
}

func (f *TableFormatter) formatSchemaDef(s *schema.Def) ([]byte, error) {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Property", "Value"})

	t.AppendRow(table.Row{"Name", s.Name})
	t.AppendRow(table.Row{"Mode", s.Mode})
	t.AppendRow(table.Row{"Version", s.Version})

	// Fields
	if len(s.Fields) > 0 {
		t.AppendSeparator()
		t.AppendRow(table.Row{"Fields", ""})
		for _, field := range s.Fields {
			t.AppendRow(table.Row{
				"  " + field.Name,
				field.Type.String(),
			})
		}
	}

	return []byte(t.Render()), nil
}

func (f *TableFormatter) formatSchemaDefs(schemaDefs []*schema.Def) ([]byte, error) {
	if len(schemaDefs) == 0 {
		return []byte("No schemas found\n"), nil
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Name", "Mode", "Fields"})

	for _, s := range schemaDefs {
		t.AppendRow(table.Row{
			s.Name,
			s.Mode,
			len(s.Fields),
		})
	}

	return []byte(t.Render()), nil
}

func (f *TableFormatter) formatMap(m map[string]any) ([]byte, error) {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"Key", "Value"})

	for k, v := range m {
		t.AppendRow(table.Row{k, fmt.Sprintf("%v", v)})
	}

	return []byte(t.Render()), nil
}

func (f *TableFormatter) ContentType() string {
	return "text/plain"
}

// OutputWriter manages output formatting and destination.
type OutputWriter struct {
	out       io.Writer
	formatter Formatter
	quiet     bool
}

// NewOutputWriter creates a writer with the specified format.
func NewOutputWriter(out io.Writer, format Format) *OutputWriter {
	var formatter Formatter

	switch format {
	case FormatJSON:
		formatter = &JSONFormatter{}
	case FormatTable:
		formatter = &TableFormatter{}
	case FormatYAML:
		formatter = &YAMLFormatter{}
	default:
		formatter = &JSONFormatter{}
	}

	return &OutputWriter{
		out:       out,
		formatter: formatter,
	}
}

// Write formats and writes data to the output.
func (w *OutputWriter) Write(data any) error {
	if w.quiet {
		return nil
	}

	formatted, err := w.formatter.Format(data)
	if err != nil {
		return fmt.Errorf("[output] formatting failed: %w", err)
	}

	_, err = w.out.Write(formatted)
	if err != nil {
		return fmt.Errorf("[output] write failed: %w", err)
	}

	_, err = w.out.Write([]byte("\n"))
	return err
}

// SetQuiet enables/disables output suppression.
func (w *OutputWriter) SetQuiet(quiet bool) {
	w.quiet = quiet
}
