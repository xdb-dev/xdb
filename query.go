package xdb

import (
	"fmt"
	"strings"
)

const (
	OpEq     = "=="
	OpNe     = "!="
	OpGt     = ">"
	OpGe     = ">="
	OpLt     = "<"
	OpLe     = "<="
	OpIn     = "IN"
	OpNotIn  = "NOT IN"
	OpSearch = "SEARCH"
	ASC      = "ASC"
	DESC     = "DESC"
)

// Query is a commmon abstraction for querying, filtering, and sorting records.
type Query struct {
	attrs    []string
	kind     string
	conds    map[string][]*Condition
	order_by []string
	skip     int
	limit    int
}

// Select creates a new query to fetch records with the given attributes.
// Example:
//
//	q := Select("name", "age")
//	q := Select("*") // fetch all attributes
func Select(attrs ...string) *Query {
	if len(attrs) > 0 && attrs[0] == "*" {
		attrs = nil
	}

	return &Query{attrs: attrs}
}

// From sets the kind of the query.
func (q *Query) From(kind string) *Query {
	q.kind = kind
	return q
}

// Where adds a filter to the query.
func (q *Query) Where(attr string) *Condition {
	if q.conds == nil {
		q.conds = make(map[string][]*Condition)
	}

	c := &Condition{q: q}

	if _, ok := q.conds[attr]; !ok {
		q.conds[attr] = make([]*Condition, 0)
	}

	q.conds[attr] = append(q.conds[attr], c)

	return c
}

// Kind returns the record kind of the query.
func (q *Query) Kind() string {
	return q.kind
}

// GetAttributes returns the attributes of the query.
func (q *Query) GetAttributes() []string {
	return q.attrs
}

// GetConditions returns the conditions of the query.
func (q *Query) GetConditions() map[string][]*Condition {
	return q.conds
}

// GetOrderBy returns the sorting order of the query.
func (q *Query) GetOrderBy() []string {
	return q.order_by
}

// GetSkip returns the offset of the query.
func (q *Query) GetSkip() int {
	return q.skip
}

// GetLimit returns the limit of the query.
func (q *Query) GetLimit() int {
	return q.limit
}

// And adds a logical AND filter to the query.
func (q *Query) And(field string) *Condition {
	if len(q.conds) == 0 {
		panic("xdb: no conditions to AND")
	}

	return q.Where(field)
}

// OrderBy adds a sorting order to the query.
// Accepts pairs of attribute and direction.
// Example:
//
//	q.OrderBy("name", "asc")
//	q.OrderBy("age", "desc")
func (q *Query) OrderBy(args ...string) *Query {
	if len(args)%2 != 0 {
		panic("xdb: OrderBy requires pairs of attribute and direction")
	}

	q.order_by = append(q.order_by, args...)

	return q
}

// Skip adds an offset to the query.
func (q *Query) Skip(offset int) *Query {
	q.skip = offset

	return q
}

// Limit adds a limit to the query.
func (q *Query) Limit(limit int) *Query {
	q.limit = limit

	return q
}

// String returns a string representation of the query.
func (q *Query) String() string {
	var sb strings.Builder

	sb.WriteString("SELECT ")

	if len(q.attrs) == 0 {
		sb.WriteString("*")
	} else {
		for i, attr := range q.attrs {
			if i > 0 {
				sb.WriteString(", ")
			}

			sb.WriteString(attr)
		}
	}

	sb.WriteString(" FROM ")
	sb.WriteString(q.kind)

	if len(q.conds) > 0 {
		sb.WriteString(" WHERE ")

		i := 0
		for attr, conds := range q.conds {
			for _, c := range conds {
				if i > 0 {
					sb.WriteString(" AND ")
				}

				sb.WriteString(attr)
				sb.WriteString(" ")
				sb.WriteString(c.op)

				if len(c.vals) > 0 {
					sb.WriteString(" (")

					for j, val := range c.vals {
						if j > 0 {
							sb.WriteString(", ")
						}

						sb.WriteString(fmt.Sprintf("'%v'", val))
					}

					sb.WriteString(")")
				} else {
					sb.WriteString(" ")
					sb.WriteString(fmt.Sprintf("'%v'", c.val))
				}

				i++
			}
		}
	}

	if len(q.order_by) > 0 {
		sb.WriteString(" ORDER BY ")

		for i := 0; i < len(q.order_by); i += 2 {
			if i > 0 {
				sb.WriteString(", ")
			}

			sb.WriteString(q.order_by[i])
			sb.WriteString(" ")
			sb.WriteString(q.order_by[i+1])
		}
	}

	if q.skip > 0 {
		sb.WriteString(" OFFSET ")
		sb.WriteString(fmt.Sprintf("%d", q.skip))
	}

	if q.limit > 0 {
		sb.WriteString(" LIMIT ")
		sb.WriteString(fmt.Sprintf("%d", q.limit))
	}

	return sb.String()
}

// Condition defines an operator and a value to filter records.
type Condition struct {
	q    *Query
	op   string
	val  any
	vals []any
}

// Op returns the operator of the condition.
func (c *Condition) Op() string {
	return c.op
}

// Value returns the value of the condition.
func (c *Condition) Value() any {
	return c.val
}

// ValueList returns the list of values of the condition.
func (c *Condition) ValueList() []any {
	return c.vals
}

func (c *Condition) setOp(op string, val any) *Query {
	if c.op != "" {
		panic("xdb: operator already set")
	}

	c.op = op
	c.val = val

	return c.q
}

func (c *Condition) setOpList(op string, vals []any) *Query {
	if c.op != "" {
		panic("xdb: operator already set")
	}

	c.op = op
	c.vals = vals

	return c.q
}

// Eq adds an equality filter to the condition.
func (c *Condition) Eq(val any) *Query {
	return c.setOp(OpEq, val)
}

// Ne adds a non-equality filter to the condition.
func (c *Condition) Ne(val any) *Query {
	return c.setOp(OpNe, val)
}

// Gt adds a greater-than filter to the condition.
func (c *Condition) Gt(val any) *Query {
	return c.setOp(OpGt, val)
}

// Ge adds a greater-than-or-equal filter to the condition.
func (c *Condition) Ge(val any) *Query {
	return c.setOp(OpGe, val)
}

// Lt adds a less-than filter to the condition.
func (c *Condition) Lt(val any) *Query {
	return c.setOp(OpLt, val)
}

// Le adds a less-than-or-equal filter to the condition.
func (c *Condition) Le(val any) *Query {
	return c.setOp(OpLe, val)
}

// In adds a set membership filter to the condition.
func (c *Condition) In(vals ...any) *Query {
	return c.setOpList(OpIn, vals)
}

// NotIn adds a set non-membership filter to the condition.
func (c *Condition) NotIn(vals ...any) *Query {
	return c.setOpList(OpNotIn, vals)
}

// Search adds a full-text search filter to the condition.
func (c *Condition) Search(val any) *Query {
	return c.setOp(OpSearch, val)
}

// ResultSet is a paginated list of T.
type ResultSet[T any] struct {
	t      []*T
	offset int
	limit  int
}

// List returns the list of T.
func (rs *ResultSet[T]) List() []*T {
	return rs.t
}

// Next returns the next offset.
func (rs *ResultSet[T]) Next() int {
	return rs.offset + rs.limit
}

// Page returns the current page number.
func (rs *ResultSet[T]) Page() int {
	return rs.offset/rs.limit + 1
}

// NewResultSet creates a new ResultSet.
func NewResultSet[T any](t []*T, offset, limit int) *ResultSet[T] {
	return &ResultSet[T]{t: t, offset: offset, limit: limit}
}
