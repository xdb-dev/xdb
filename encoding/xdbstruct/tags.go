package xdbstruct

import (
	"strings"
)

// field represents parsed struct field information.
type field struct {
	name       string
	skip       bool
	primaryKey bool
	value      any
}

// parseTag parses a struct tag value into a field.
// Tag format: "fieldName,option1,option2,..."
// Special cases:
//   - "-" means skip this field
//   - "primary_key" option marks the field as the record's ID
func parseTag(tag string) field {
	if tag == "" {
		return field{}
	}

	parts := strings.Split(tag, ",")

	ft := field{
		name: strings.TrimSpace(parts[0]),
	}

	if ft.name == "-" {
		ft.skip = true
		return ft
	}

	for i := 1; i < len(parts); i++ {
		option := strings.TrimSpace(parts[i])
		switch option {
		case "primary_key":
			ft.primaryKey = true
		}
	}

	return ft
}
