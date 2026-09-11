package output

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// writeDef writes one [schema.Def] as key/value rows. Each field takes
// one row under "fields". The fields of an object array follow their
// parent, indented.
func writeDef(w io.Writer, def *schema.Def) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	rows := [][2]string{
		{"uri", def.URI.String()},
	}

	if def.Description != "" {
		rows = append(rows, [2]string{"description", def.Description})
	}

	rows = append(rows,
		[2]string{"mode", string(def.Mode)},
		[2]string{"revision", strconv.FormatInt(def.Revision, 10)},
	)

	for _, r := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", r[0], r[1]); err != nil {
			return err
		}
	}

	label := "fields"
	for _, r := range fieldRows(def.Fields, "") {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", label, r[0], r[1]); err != nil {
			return err
		}

		label = ""
	}

	return tw.Flush()
}

// writeDefList writes one row per [schema.Def]: the URI, the mode, the
// revision, and the number of fields.
func writeDefList(w io.Writer, items []any) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	if err := writeRow(tw, []string{"uri", "mode", "revision", "fields"}); err != nil {
		return err
	}

	for _, item := range items {
		def := item.(*schema.Def)
		cols := []string{
			def.URI.String(),
			string(def.Mode),
			strconv.FormatInt(def.Revision, 10),
			strconv.Itoa(len(def.Fields)),
		}

		if err := writeRow(tw, cols); err != nil {
			return err
		}
	}

	return tw.Flush()
}

// fieldRows returns the name and the summary of each field. User fields
// come first and system fields last, each group sorted by name. The
// fields of an object array follow their parent, indented by two spaces.
func fieldRows(fields map[string]schema.Field, indent string) [][2]string {
	rows := make([][2]string, 0, len(fields))

	for _, name := range fieldOrder(fields) {
		field := fields[name]
		rows = append(rows, [2]string{indent + name, fieldSummary(field)})
		rows = append(rows, fieldRows(field.Items, indent+"  ")...)
	}

	return rows
}

// fieldOrder returns the field names sorted, with system fields last.
func fieldOrder(fields map[string]schema.Field) []string {
	names := slices.Collect(maps.Keys(fields))

	slices.SortFunc(names, func(a, b string) int {
		aSystem := schema.IsSystemField(a)
		bSystem := schema.IsSystemField(b)

		if aSystem != bSystem {
			if aSystem {
				return 1
			}

			return -1
		}

		return strings.Compare(a, b)
	})

	return names
}

// fieldSummary returns the type of a field and its markers, for example
// "string (required, unique)".
func fieldSummary(f schema.Field) string {
	typeName := f.Type.ID().Lower()
	if f.Type.ID() == core.TIDArray && f.Type.ElemTypeID() != "" {
		typeName = "array<" + f.Type.ElemTypeID().Lower() + ">"
	}

	var markers []string

	if f.Required {
		markers = append(markers, "required")
	}

	if f.Indexed {
		markers = append(markers, "indexed")
	}

	if f.Unique {
		markers = append(markers, "unique")
	}

	if len(markers) == 0 {
		return typeName
	}

	return typeName + " (" + strings.Join(markers, ", ") + ")"
}
