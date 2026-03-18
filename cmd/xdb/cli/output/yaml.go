package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

type yamlFormatter struct{}

func (f *yamlFormatter) FormatOne(w io.Writer, v any) error {
	if _, err := io.WriteString(w, "---\n"); err != nil {
		return err
	}

	return yaml.NewEncoder(w).Encode(v)
}

func (f *yamlFormatter) FormatList(w io.Writer, items []any) error {
	if _, err := io.WriteString(w, "---\n"); err != nil {
		return err
	}

	if items == nil {
		items = []any{}
	}

	return yaml.NewEncoder(w).Encode(items)
}

func (f *yamlFormatter) FormatError(w io.Writer, err error) error {
	if _, werr := io.WriteString(w, "---\n"); werr != nil {
		return werr
	}

	return yaml.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
