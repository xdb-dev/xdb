package filter

import (
	"fmt"
	"strings"
)

// Op represents a comparison operator.
type Op string

const (
	// OpEq is the equality operator (=).
	OpEq Op = "="
	// OpNe is the not-equal operator (!=).
	OpNe Op = "!="
	// OpGt is the greater-than operator (>).
	OpGt Op = ">"
	// OpLt is the less-than operator (<).
	OpLt Op = "<"
	// OpGte is the greater-than-or-equal operator (>=).
	OpGte Op = ">="
	// OpLte is the less-than-or-equal operator (<=).
	OpLte Op = "<="
	// OpContains is the substring containment operator.
	OpContains Op = "contains"
)

// Expr is a parsed filter expression.
type Expr struct {
	Attr  string
	Op    Op
	Value string
}

// operators lists the operators in order of longest first so that >= is
// matched before >, etc.
var operators = []struct {
	token string
	op    Op
}{
	{">=", OpGte},
	{"<=", OpLte},
	{"!=", OpNe},
	{" contains ", OpContains},
	{">", OpGt},
	{"<", OpLt},
	{"=", OpEq},
}

// Parse parses a filter expression string into an [Expr].
func Parse(s string) (*Expr, error) {
	if s == "" {
		return nil, fmt.Errorf("filter: empty expression")
	}

	for _, op := range operators {
		idx := strings.Index(s, op.token)
		if idx < 0 {
			continue
		}

		attr := strings.TrimSpace(s[:idx])
		value := strings.TrimSpace(s[idx+len(op.token):])

		if attr == "" {
			return nil, fmt.Errorf("filter: empty attribute")
		}

		// Strip surrounding quotes from value.
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		if value == "" {
			return nil, fmt.Errorf("filter: empty value")
		}

		return &Expr{
			Attr:  attr,
			Op:    op.op,
			Value: value,
		}, nil
	}

	return nil, fmt.Errorf("filter: no operator found in %q", s)
}
