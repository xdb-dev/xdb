package output

import (
	"fmt"
	"io"
)

type tableFormatter struct{}

func (f *tableFormatter) FormatOne(_ io.Writer, _ any) error {
	return fmt.Errorf("output: table FormatOne not implemented")
}

func (f *tableFormatter) FormatList(_ io.Writer, _ []any) error {
	return fmt.Errorf("output: table FormatList not implemented")
}

func (f *tableFormatter) FormatError(_ io.Writer, _ error) error {
	return fmt.Errorf("output: table FormatError not implemented")
}
