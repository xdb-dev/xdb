package filter

import (
	"strconv"
	"strings"

	"github.com/xdb-dev/xdb/core"
)

// Match evaluates an [Expr] against a [core.Record].
// Returns true if the record matches the filter.
func Match(expr *Expr, record *core.Record) bool {
	tuple := record.Get(expr.Attr)
	if tuple == nil {
		return false
	}

	recordValue := tuple.Value().String()

	switch expr.Op {
	case OpEq:
		return recordValue == expr.Value
	case OpNe:
		return recordValue != expr.Value
	case OpContains:
		return strings.Contains(recordValue, expr.Value)
	case OpGt, OpLt, OpGte, OpLte:
		return compareOrdered(recordValue, expr.Value, expr.Op)
	default:
		return false
	}
}

// compareOrdered compares two values using the given ordering operator.
// If both values parse as floats, comparison is numeric; otherwise lexicographic.
func compareOrdered(recordValue, filterValue string, op Op) bool {
	rv, rerr := strconv.ParseFloat(recordValue, 64)
	fv, ferr := strconv.ParseFloat(filterValue, 64)

	if rerr == nil && ferr == nil {
		return compareFloat(rv, fv, op)
	}

	return compareString(recordValue, filterValue, op)
}

func compareFloat(a, b float64, op Op) bool {
	switch op {
	case OpGt:
		return a > b
	case OpLt:
		return a < b
	case OpGte:
		return a >= b
	case OpLte:
		return a <= b
	default:
		return false
	}
}

func compareString(a, b string, op Op) bool {
	switch op {
	case OpGt:
		return a > b
	case OpLt:
		return a < b
	case OpGte:
		return a >= b
	case OpLte:
		return a <= b
	default:
		return false
	}
}
