package store

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// descPrefix marks a descending field in [Query.OrderBy].
const descPrefix = "-"

// reservedOrderFields are the reserved fields that every list can sort by.
// A def can declare them too, as a stored def does for _version and
// _updated.
var reservedOrderFields = []string{
	schema.FieldID,
	schema.FieldVersion,
	schema.FieldUpdated,
}

// OrderKey is one field of a list order.
type OrderKey struct {
	Field string
	Desc  bool
}

// ParseOrderBy parses the fields of [Query.OrderBy]. A leading "-" sorts a
// field descending. An empty field name is [core.ErrInvalidOrder].
func ParseOrderBy(fields []string) ([]OrderKey, error) {
	keys := make([]OrderKey, 0, len(fields))

	for _, raw := range fields {
		name, desc := strings.CutPrefix(raw, descPrefix)
		if name == "" {
			return nil, fmt.Errorf("%w: empty field name in order_by entry %q", core.ErrInvalidOrder, raw)
		}

		keys = append(keys, OrderKey{
			Field: name,
			Desc:  desc,
		})
	}

	return keys, nil
}

// checkOrderFields rejects an order key that a strict def does not declare.
// The reserved fields are always known. A flexible or dynamic def accepts
// every name, as [filter.Compile] does, because its records can hold
// undeclared attributes. A nil def, which is a namespace scope or a
// schema-free table, also accepts every name.
func checkOrderFields(keys []OrderKey, def *schema.Def) error {
	if def == nil || def.Mode != schema.ModeStrict {
		return nil
	}

	for _, key := range keys {
		if isOrderable(key.Field, def) {
			continue
		}

		names := slices.Collect(maps.Keys(def.Fields))
		for _, name := range reservedOrderFields {
			if _, declared := def.Fields[name]; !declared {
				names = append(names, name)
			}
		}
		slices.Sort(names)

		return fmt.Errorf("%w: unknown field %q in order_by; available fields: %s",
			core.ErrInvalidOrder,
			key.Field,
			strings.Join(names, ", "),
		)
	}

	return nil
}

// isOrderable reports whether def declares name, or name is a reserved
// field.
func isOrderable(name string, def *schema.Def) bool {
	if _, ok := def.Fields[name]; ok {
		return true
	}

	return slices.Contains(reservedOrderFields, name)
}

// sortRecords sorts records by keys, then by path. A missing value sorts
// first ascending and last descending, which is where SQLite puts a NULL,
// so a driver that pushes the order down to SQL returns the same order.
// The path breaks every tie, so pages of the same order are stable.
func sortRecords(records []*core.Record, keys []OrderKey) {
	slices.SortStableFunc(records, func(a, b *core.Record) int {
		for _, key := range keys {
			c := compareValues(attrValue(a, key.Field), attrValue(b, key.Field))
			if key.Desc {
				c = -c
			}

			if c != 0 {
				return c
			}
		}

		return strings.Compare(a.URI().Path(), b.URI().Path())
	})
}

// attrValue returns the value of attr in record, or nil when it is absent.
func attrValue(record *core.Record, attr string) *core.Value {
	tuple := record.Get(attr)
	if tuple == nil {
		return nil
	}

	return tuple.Value()
}

// compareValues orders two values. A nil value comes first. Values of two
// types order by the name of the type, so a mixed column still has one
// order. A type with no natural order compares as text.
func compareValues(a, b *core.Value) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	}

	ta := a.Type().ID()
	tb := b.Type().ID()

	if ta != tb {
		return strings.Compare(ta.String(), tb.String())
	}

	switch ta {
	case core.TIDString:
		return compareAs(a, b, (*core.Value).AsStr, strings.Compare)
	case core.TIDInteger:
		return compareAs(a, b, (*core.Value).AsInt, cmp.Compare[int64])
	case core.TIDUnsigned:
		return compareAs(a, b, (*core.Value).AsUint, cmp.Compare[uint64])
	case core.TIDFloat:
		return compareAs(a, b, (*core.Value).AsFloat, cmp.Compare[float64])
	case core.TIDBoolean:
		return compareAs(a, b, (*core.Value).AsBool, compareBool)
	case core.TIDTime:
		av, aErr := a.AsTime()
		bv, bErr := b.AsTime()
		// SQLite stores a time in milliseconds. A comparison at a finer
		// precision gives a different order from the SQL pushdown.
		if aErr == nil && bErr == nil {
			return cmp.Compare(av.UnixMilli(), bv.UnixMilli())
		}
	}

	return strings.Compare(a.String(), b.String())
}

// compareAs reads both values with as and compares them with compare. A
// value that does not read compares as text.
func compareAs[T any](
	a *core.Value,
	b *core.Value,
	as func(*core.Value) (T, error),
	compare func(T, T) int,
) int {
	av, aErr := as(a)
	bv, bErr := as(b)

	if aErr != nil || bErr != nil {
		return strings.Compare(a.String(), b.String())
	}

	return compare(av, bv)
}

// compareBool orders false before true.
func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	default:
		return 1
	}
}
