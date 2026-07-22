package output

import (
	"encoding/json"
	"io"
)

type jsonFormatter struct{}

func (f *jsonFormatter) FormatOne(w io.Writer, v any) error {
	return writeIndentedJSON(w, v)
}

func (f *jsonFormatter) FormatList(w io.Writer, items []any) error {
	if items == nil {
		items = []any{}
	}

	return writeIndentedJSON(w, items)
}

func (f *jsonFormatter) FormatPage(w io.Writer, p Page) error {
	return writeIndentedJSON(w, p.doc())
}

func (f *jsonFormatter) FormatError(w io.Writer, err error) error {
	if env, ok := err.(*ErrorEnvelope); ok {
		return writeIndentedJSON(w, env)
	}

	return writeIndentedJSON(w, map[string]string{"error": err.Error()})
}

type ndjsonFormatter struct{}

func (f *ndjsonFormatter) FormatOne(w io.Writer, v any) error {
	return writeCompactJSON(w, v)
}

func (f *ndjsonFormatter) FormatList(w io.Writer, items []any) error {
	for _, item := range items {
		if err := writeCompactJSON(w, item); err != nil {
			return err
		}
	}

	return nil
}

func (f *ndjsonFormatter) FormatPage(w io.Writer, p Page) error {
	return f.FormatList(w, p.Items)
}

func (f *ndjsonFormatter) FormatError(w io.Writer, err error) error {
	if env, ok := err.(*ErrorEnvelope); ok {
		return writeCompactJSON(w, env)
	}

	return writeCompactJSON(w, map[string]string{"error": err.Error()})
}

func writeIndentedJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	return enc.Encode(v)
}

func writeCompactJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	return enc.Encode(v)
}
