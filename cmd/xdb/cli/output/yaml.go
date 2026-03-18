package output

import (
	"fmt"
	"io"
)

type yamlFormatter struct{}

func (f *yamlFormatter) FormatOne(_ io.Writer, _ any) error {
	return fmt.Errorf("output: yaml FormatOne not implemented")
}

func (f *yamlFormatter) FormatList(_ io.Writer, _ []any) error {
	return fmt.Errorf("output: yaml FormatList not implemented")
}

func (f *yamlFormatter) FormatError(_ io.Writer, _ error) error {
	return fmt.Errorf("output: yaml FormatError not implemented")
}
