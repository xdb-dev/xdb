package output

import (
	"encoding/json"
	"io"

	"gopkg.in/yaml.v3"
)

type yamlFormatter struct{}

func (f *yamlFormatter) FormatOne(w io.Writer, v any) error {
	if _, err := io.WriteString(w, "---\n"); err != nil {
		return err
	}

	return encodeYAML(w, v)
}

func (f *yamlFormatter) FormatList(w io.Writer, items []any) error {
	if _, err := io.WriteString(w, "---\n"); err != nil {
		return err
	}

	if items == nil {
		items = []any{}
	}

	return encodeYAML(w, items)
}

func (f *yamlFormatter) FormatPage(w io.Writer, p Page) error {
	return f.FormatOne(w, p.doc())
}

func (f *yamlFormatter) FormatError(w io.Writer, err error) error {
	if _, werr := io.WriteString(w, "---\n"); werr != nil {
		return werr
	}

	if env, ok := err.(*ErrorEnvelope); ok {
		return encodeYAML(w, env)
	}

	return encodeYAML(w, map[string]string{"error": err.Error()})
}

// encodeYAML writes the YAML form of the JSON document of v. YAML is a
// superset of JSON, so the JSON is read into a node tree and written
// again as YAML. A value then carries its json tags and the output of
// its [json.Marshaler] into both formats, and the keys keep the order
// that JSON gave them.
func encodeYAML(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	plainStyle(&doc)

	return yaml.NewEncoder(w).Encode(&doc)
}

// plainStyle removes the flow style and the quotes that the JSON syntax
// leaves on each node. The encoder then writes block YAML, and it quotes
// only the scalars that need quotes.
func plainStyle(n *yaml.Node) {
	n.Style = 0

	for _, child := range n.Content {
		plainStyle(child)
	}
}
