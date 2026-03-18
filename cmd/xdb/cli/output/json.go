package output

import (
	"fmt"
	"io"
)

type jsonFormatter struct{}

func (f *jsonFormatter) FormatOne(_ io.Writer, _ any) error {
	return fmt.Errorf("output: json FormatOne not implemented")
}

func (f *jsonFormatter) FormatList(_ io.Writer, _ []any) error {
	return fmt.Errorf("output: json FormatList not implemented")
}

func (f *jsonFormatter) FormatError(_ io.Writer, _ error) error {
	return fmt.Errorf("output: json FormatError not implemented")
}

type ndjsonFormatter struct{}

func (f *ndjsonFormatter) FormatOne(_ io.Writer, _ any) error {
	return fmt.Errorf("output: ndjson FormatOne not implemented")
}

func (f *ndjsonFormatter) FormatList(_ io.Writer, _ []any) error {
	return fmt.Errorf("output: ndjson FormatList not implemented")
}

func (f *ndjsonFormatter) FormatError(_ io.Writer, _ error) error {
	return fmt.Errorf("output: ndjson FormatError not implemented")
}
