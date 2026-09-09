package sqlgen

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/schema"
)

// ErrUnknownColumn is returned by [Generate] under [ColumnStrategy] when a
// filter references an identifier that is not among the compiled filter's
// schema fields. Callers map it to a query-pushdown refusal so the record
// store can fall back to an in-memory scan.
var ErrUnknownColumn = errors.New("[xdb/sqlgen] unknown column")

// Strategy identifies the SQL table layout.
type Strategy int

const (
	// ColumnStrategy generates SQL for tables with one column per field.
	ColumnStrategy Strategy = iota

	// KVStrategy generates SQL for entity-attribute-value tables
	// with (_id, _attr, _type, _val) rows.
	KVStrategy
)

// WhereClause holds a parameterized SQL WHERE fragment.
type WhereClause struct {
	SQL    string
	Params []any
}

// Generate walks the CEL AST of a compiled [filter.Filter] and produces
// a parameterized SQL WHERE clause. For [KVStrategy], table is used in
// subqueries and must not be empty.
func Generate(f *filter.Filter, strategy Strategy, table string) (*WhereClause, error) {
	nativeAst := f.CelAst().NativeRep()
	g := &generator{
		strategy: strategy,
		table:    table,
		typeMap:  nativeAst.TypeMap(),
		def:      f.Def(),
	}

	sql, err := g.walk(nativeAst.Expr())
	if err != nil {
		return nil, err
	}

	return &WhereClause{
		SQL:    sql,
		Params: g.params,
	}, nil
}

// generator walks the CEL AST and accumulates SQL and params.
type generator struct {
	typeMap  map[int64]*types.Type
	def      *schema.Def
	table    string
	params   []any
	strategy Strategy
}

// walk recursively converts a CEL AST node to SQL.
func (g *generator) walk(expr ast.Expr) (string, error) {
	switch expr.Kind() {
	case ast.CallKind:
		return g.walkCall(expr)
	case ast.IdentKind:
		return g.walkIdent(expr.AsIdent())
	case ast.LiteralKind:
		return g.walkLiteral(expr.AsLiteral())
	case ast.SelectKind:
		return g.walkSelect(expr)
	case ast.ListKind:
		return g.walkList(expr)
	default:
		return "", fmt.Errorf("[xdb/sqlgen] unsupported expression kind: %v", expr.Kind())
	}
}

// walkIdent resolves a top-level identifier. Under [KVStrategy], attribute
// names are bound as query parameters, not SQL identifiers, so the raw name
// passes through untouched. Under [ColumnStrategy], the name IS a SQL
// column reference: it must be one of the compiled filter's schema fields
// (nil def trusts the ident, matching filter.Compile's own nil-def
// permissiveness), and is double-quoted for emission.
func (g *generator) walkIdent(name string) (string, error) {
	if g.strategy == KVStrategy {
		return name, nil
	}

	// The record id is projected from the path, never declared as a
	// field — but it is exactly the physical id column both layouts key
	// on, so it needs an exemption from the membership check and no
	// translation beyond it.
	if name == schema.FieldID {
		return quoteIdent(name), nil
	}

	if g.def != nil {
		if _, ok := g.def.Fields[name]; !ok {
			return "", fmt.Errorf("%w: %q", ErrUnknownColumn, name)
		}
	}

	return quoteIdent(name), nil
}

// quoteIdent double-quotes a SQL identifier, doubling any embedded quotes.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// walkCall handles operator and function call expressions.
func (g *generator) walkCall(expr ast.Expr) (string, error) {
	call := expr.AsCall()
	fn := call.FunctionName()
	args := call.Args()

	switch fn {
	// Binary comparison operators.
	case "_==_", "_!=_", "_<_", "_>_", "_<=_", "_>=_":
		return g.walkBinaryOp(fn, args)

	// Logical operators.
	case "_&&_":
		return g.walkLogical("AND", args)
	case "_||_":
		return g.walkLogical("OR", args)
	case "!_":
		return g.walkNot(args)

	// String member functions.
	case "contains":
		return g.walkStringFn(call, "contains")
	case "startsWith":
		return g.walkStringFn(call, "startsWith")
	case "endsWith":
		return g.walkStringFn(call, "endsWith")

	// Global functions.
	case "size":
		return g.walkSize(args)

	// List membership.
	case "@in":
		return g.walkIn(args)

	default:
		return "", fmt.Errorf("[xdb/sqlgen] unsupported function: %s", fn)
	}
}

// walkBinaryOp handles ==, !=, <, >, <=, >=.
func (g *generator) walkBinaryOp(fn string, args []ast.Expr) (string, error) {
	sqlOp := celToSQLOp(fn)

	if g.strategy == KVStrategy {
		return g.walkKVComparison(args[0], sqlOp, args[1])
	}

	left, err := g.walk(args[0])
	if err != nil {
		return "", err
	}

	right, err := g.walk(args[1])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", left, sqlOp, right), nil
}

// walkLogical handles AND / OR.
func (g *generator) walkLogical(op string, args []ast.Expr) (string, error) {
	left, err := g.walk(args[0])
	if err != nil {
		return "", err
	}

	right, err := g.walk(args[1])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", left, op, right), nil
}

// walkNot handles logical NOT.
func (g *generator) walkNot(args []ast.Expr) (string, error) {
	inner, err := g.walk(args[0])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(NOT %s)", inner), nil
}

// walkStringFn handles contains, startsWith, endsWith as member functions.
func (g *generator) walkStringFn(call ast.CallExpr, fn string) (string, error) {
	target, err := g.walk(call.Target())
	if err != nil {
		return "", err
	}

	if len(call.Args()) != 1 {
		return "", fmt.Errorf("[xdb/sqlgen] %s expects 1 argument", fn)
	}

	argVal := call.Args()[0].AsLiteral()
	g.params = append(g.params, argVal.Value())

	if g.strategy == KVStrategy {
		return g.walkKVStringFn(target, fn)
	}

	// contains/startsWith/endsWith use instr/substr rather than LIKE:
	// SQLite's LIKE is ASCII case-insensitive by default, which diverges
	// from CEL's byte-wise, case-sensitive string semantics.
	switch fn {
	case "contains":
		return fmt.Sprintf("(instr(%s, ?) > 0)", target), nil
	case "startsWith":
		g.params = append(g.params, argVal.Value())
		return fmt.Sprintf("(substr(%s, 1, length(?)) = ?)", target), nil
	case "endsWith":
		g.params = append(g.params, argVal.Value())
		return fmt.Sprintf("(substr(%s, -length(?)) = ?)", target), nil
	default:
		return "", fmt.Errorf("[xdb/sqlgen] unsupported string function: %s", fn)
	}
}

// walkSize handles size(field) -> LENGTH(field).
func (g *generator) walkSize(args []ast.Expr) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("[xdb/sqlgen] size expects 1 argument")
	}

	inner, err := g.walk(args[0])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("LENGTH(%s)", inner), nil
}

// walkIn handles x in [a, b, c] -> x IN (?, ?, ?).
func (g *generator) walkIn(args []ast.Expr) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("[xdb/sqlgen] in expects 2 arguments")
	}

	field, err := g.walk(args[0])
	if err != nil {
		return "", err
	}

	// The KV subquery binds the attr name before the list values, so it
	// must be appended before them. Prepending it to g.params instead
	// would shift every placeholder bound by an earlier clause.
	if g.strategy == KVStrategy {
		g.params = append(g.params, field)
	}

	list := args[1].AsList()
	placeholders := make([]string, len(list.Elements()))
	for i, elem := range list.Elements() {
		val := elem.AsLiteral()
		g.params = append(g.params, val.Value())
		placeholders[i] = "?"
	}

	if g.strategy == KVStrategy {
		inner := strings.Join(placeholders, ", ")
		return fmt.Sprintf("(_id IN (SELECT _id FROM %s WHERE _attr = ? AND CAST(_val AS TEXT) IN (%s)))",
			g.table, inner), nil
	}

	return fmt.Sprintf("(%s IN (%s))", field, strings.Join(placeholders, ", ")), nil
}

// walkLiteral converts a CEL literal to a SQL placeholder.
func (g *generator) walkLiteral(val ref.Val) (string, error) {
	g.params = append(g.params, val.Value())
	return "?", nil
}

// walkSelect handles field selection (e.g., author.name).
func (g *generator) walkSelect(expr ast.Expr) (string, error) {
	sel := expr.AsSelect()
	operand, err := g.walk(sel.Operand())
	if err != nil {
		return "", err
	}

	return operand + "." + sel.FieldName(), nil
}

// walkList rejects a standalone list literal. walkIn consumes the list
// argument of @in directly, so a list that reaches walkList is not an @in
// argument and has no SQL form.
func (g *generator) walkList(expr ast.Expr) (string, error) {
	return "", fmt.Errorf("[xdb/sqlgen] unexpected standalone list expression")
}

// --- KV strategy helpers ---

// walkKVComparison generates a KV subquery for a binary comparison.
func (g *generator) walkKVComparison(
	left ast.Expr,
	sqlOp string,
	right ast.Expr,
) (string, error) {
	attrName, err := g.resolveAttrName(left)
	if err != nil {
		return "", err
	}

	val := right.AsLiteral()
	g.params = append(g.params, attrName)

	// Determine if we need CAST for numeric comparison.
	valCast := g.kvValExpr(val)
	g.params = append(g.params, val.Value())

	return fmt.Sprintf("(_id IN (SELECT _id FROM %s WHERE _attr = ? AND %s %s ?))",
		g.table, valCast, sqlOp), nil
}

// walkKVStringFn generates a KV subquery for a string function. Like the
// column-strategy path, it uses instr/substr rather than LIKE for
// case-sensitive, CEL-equivalent matching.
func (g *generator) walkKVStringFn(attrName, fn string) (string, error) {
	// The arg value was already added to params by the caller.
	// We need to add the attr name before it.
	argVal := g.params[len(g.params)-1]
	g.params[len(g.params)-1] = attrName
	g.params = append(g.params, argVal)

	var pattern string
	switch fn {
	case "contains":
		pattern = "instr(CAST(_val AS TEXT), ?) > 0"
	case "startsWith":
		pattern = "substr(CAST(_val AS TEXT), 1, length(?)) = ?"
		g.params = append(g.params, argVal)
	case "endsWith":
		pattern = "substr(CAST(_val AS TEXT), -length(?)) = ?"
		g.params = append(g.params, argVal)
	default:
		return "", fmt.Errorf("[xdb/sqlgen] unsupported KV string function: %s", fn)
	}

	return fmt.Sprintf("(_id IN (SELECT _id FROM %s WHERE _attr = ? AND %s))",
		g.table, pattern), nil
}

// resolveAttrName extracts the attribute name from an ident or select expression.
func (g *generator) resolveAttrName(expr ast.Expr) (string, error) {
	switch expr.Kind() {
	case ast.IdentKind:
		return expr.AsIdent(), nil
	case ast.SelectKind:
		sel := expr.AsSelect()
		parent, err := g.resolveAttrName(sel.Operand())
		if err != nil {
			return "", err
		}
		return parent + "." + sel.FieldName(), nil
	default:
		return "", fmt.Errorf("[xdb/sqlgen] cannot resolve attribute from expression kind %v", expr.Kind())
	}
}

// kvValExpr casts _val to REAL for numeric and boolean comparisons,
// or TEXT for other comparisons, based on the literal's CEL type.
func (g *generator) kvValExpr(val ref.Val) string {
	switch val.Type() {
	case types.IntType, types.UintType, types.DoubleType, types.BoolType:
		return "CAST(_val AS REAL)"
	default:
		return "CAST(_val AS TEXT)"
	}
}

// celToSQLOp maps CEL operator function names to SQL operators.
func celToSQLOp(fn string) string {
	switch fn {
	case "_==_":
		return "="
	case "_!=_":
		return "!="
	case "_<_":
		return "<"
	case "_>_":
		return ">"
	case "_<=_":
		return "<="
	case "_>=_":
		return ">="
	default:
		return fn
	}
}
