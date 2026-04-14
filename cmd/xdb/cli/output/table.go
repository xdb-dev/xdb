package output

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"text/tabwriter"
)

type tableFormatter struct{}

func (f *tableFormatter) FormatOne(w io.Writer, v any) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Map:
		keys := sortedMapKeys(rv)
		for _, k := range keys {
			val := rv.MapIndex(reflect.ValueOf(k))
			if _, err := fmt.Fprintf(tw, "%s\t%v\n", k, val.Interface()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		rt := rv.Type()
		for i := range rt.NumField() {
			name := fieldName(rt.Field(i))
			if _, err := fmt.Fprintf(tw, "%s\t%v\n", name, rv.Field(i).Interface()); err != nil {
				return err
			}
		}
	default:
		if _, err := fmt.Fprintf(tw, "%v\n", v); err != nil {
			return err
		}
	}

	return tw.Flush()
}

func (f *tableFormatter) FormatList(w io.Writer, items []any) error {
	if len(items) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	first := reflect.ValueOf(items[0])

	var err error

	switch first.Kind() {
	case reflect.Map:
		err = writeMapTable(tw, items, first)
	case reflect.Struct:
		err = writeStructTable(tw, items, first)
	default:
		for _, item := range items {
			if _, werr := fmt.Fprintf(tw, "%v\n", item); werr != nil {
				return werr
			}
		}
	}

	if err != nil {
		return err
	}

	return tw.Flush()
}

func (f *tableFormatter) FormatError(w io.Writer, err error) error {
	env, ok := err.(*ErrorEnvelope)
	if !ok {
		_, werr := fmt.Fprintf(w, "Error: %s\n", err.Error())
		return werr
	}

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	rows := [][2]string{
		{"Error", env.Code},
		{"Message", env.Message},
	}

	if env.Resource != "" {
		rows = append(rows, [2]string{"Resource", env.Resource})
	}

	if env.Action != "" {
		rows = append(rows, [2]string{"Action", env.Action})
	}

	if env.URI != "" {
		rows = append(rows, [2]string{"URI", env.URI})
	}

	if env.Hint != "" {
		rows = append(rows, [2]string{"Hint", env.Hint})
	}

	for _, r := range rows {
		if _, werr := fmt.Fprintf(tw, "%s\t%s\n", r[0], r[1]); werr != nil {
			return werr
		}
	}

	return tw.Flush()
}

func writeMapTable(tw *tabwriter.Writer, items []any, first reflect.Value) error {
	keys := sortedMapKeys(first)

	if err := writeRow(tw, keys); err != nil {
		return err
	}

	for _, item := range items {
		rv := reflect.ValueOf(item)
		vals := make([]string, 0, len(keys))

		for _, k := range keys {
			val := rv.MapIndex(reflect.ValueOf(k))
			vals = append(vals, fmt.Sprintf("%v", val.Interface()))
		}

		if err := writeRow(tw, vals); err != nil {
			return err
		}
	}

	return nil
}

func writeStructTable(tw *tabwriter.Writer, items []any, first reflect.Value) error {
	rt := first.Type()

	headers := make([]string, 0, rt.NumField())
	for i := range rt.NumField() {
		headers = append(headers, fieldName(rt.Field(i)))
	}

	if err := writeRow(tw, headers); err != nil {
		return err
	}

	for _, item := range items {
		rv := reflect.ValueOf(item)
		vals := make([]string, 0, rt.NumField())

		for i := range rt.NumField() {
			vals = append(vals, fmt.Sprintf("%v", rv.Field(i).Interface()))
		}

		if err := writeRow(tw, vals); err != nil {
			return err
		}
	}

	return nil
}

func writeRow(tw *tabwriter.Writer, cols []string) error {
	for i, col := range cols {
		if i > 0 {
			if _, err := fmt.Fprint(tw, "\t"); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprint(tw, col); err != nil {
			return err
		}
	}

	_, err := fmt.Fprint(tw, "\n")

	return err
}

func fieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag != "" && tag != "-" {
		for i := range len(tag) {
			if tag[i] == ',' {
				return tag[:i]
			}
		}

		return tag
	}

	return f.Name
}

func sortedMapKeys(rv reflect.Value) []string {
	keys := make([]string, 0, rv.Len())
	for _, k := range rv.MapKeys() {
		keys = append(keys, fmt.Sprintf("%v", k.Interface()))
	}

	sort.Strings(keys)

	return keys
}
