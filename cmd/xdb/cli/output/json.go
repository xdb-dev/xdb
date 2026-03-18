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

func (f *jsonFormatter) FormatError(w io.Writer, err error) error {
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

func (f *ndjsonFormatter) FormatError(w io.Writer, err error) error {
	return writeCompactJSON(w, map[string]string{"error": err.Error()})
}

func writeIndentedJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = w.Write(data)

	return err
}

func writeCompactJSON(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = w.Write(data)

	return err
}
