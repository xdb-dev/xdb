package sqlgen

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/xdb-dev/xdb/filter"
)

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
		return expr.AsIdent(), nil
	case ast.LiteralKind:
		return g.walkLiteral(expr.AsLiteral())
	case ast.SelectKind:
		return g.walkSelect(expr)
	case ast.ListKind:
		return g.walkList(expr)
	default:
		return "", fmt.Errorf("sqlgen: unsupported expression kind: %v", expr.Kind())
	}
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
		return "", fmt.Errorf("sqlgen: unsupported function: %s", fn)
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
		return "", fmt.Errorf("sqlgen: %s expects 1 argument", fn)
	}

	argVal := call.Args()[0].AsLiteral()
	g.params = append(g.params, argVal.Value())

	if g.strategy == KVStrategy {
		return g.walkKVStringFn(target, fn)
	}

	switch fn {
	case "contains":
		return fmt.Sprintf("(%s LIKE '%%' || ? || '%%')", target), nil
	case "startsWith":
		return fmt.Sprintf("(%s LIKE ? || '%%')", target), nil
	case "endsWith":
		return fmt.Sprintf("(%s LIKE '%%' || ?)", target), nil
	default:
		return "", fmt.Errorf("sqlgen: unsupported string function: %s", fn)
	}
}

// walkSize handles size(field) -> LENGTH(field).
func (g *generator) walkSize(args []ast.Expr) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("sqlgen: size expects 1 argument")
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
		return "", fmt.Errorf("sqlgen: in expects 2 arguments")
	}

	field, err := g.walk(args[0])
	if err != nil {
		return "", err
	}

	list := args[1].AsList()
	placeholders := make([]string, len(list.Elements()))
	for i, elem := range list.Elements() {
		val := elem.AsLiteral()
		g.params = append(g.params, val.Value())
		placeholders[i] = "?"
	}

	if g.strategy == KVStrategy {
		g.params = append([]any{field}, g.params...)
		// Rebuild: attr param first, then list values.
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

// walkList handles list literals (used as argument to @in).
func (g *generator) walkList(expr ast.Expr) (string, error) {
	// Lists are handled directly in walkIn. If we get here,
	// it's an unexpected standalone list.
	return "", fmt.Errorf("sqlgen: unexpected standalone list expression")
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

// walkKVStringFn generates a KV subquery for a string function.
func (g *generator) walkKVStringFn(attrName, fn string) (string, error) {
	// The arg value was already added to params by the caller.
	// We need to add the attr name before it.
	argVal := g.params[len(g.params)-1]
	g.params[len(g.params)-1] = attrName
	g.params = append(g.params, argVal)

	var pattern string
	switch fn {
	case "contains":
		pattern = "CAST(_val AS TEXT) LIKE '%' || ? || '%'"
	case "startsWith":
		pattern = "CAST(_val AS TEXT) LIKE ? || '%'"
	case "endsWith":
		pattern = "CAST(_val AS TEXT) LIKE '%' || ?"
	default:
		return "", fmt.Errorf("sqlgen: unsupported KV string function: %s", fn)
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
		return "", fmt.Errorf("sqlgen: cannot resolve attribute from expression kind %v", expr.Kind())
	}
}

// kvValExpr returns the SQL expression for comparing _val.
// KV tables store values as BLOB (raw bytes), so we must CAST
// to the appropriate type for comparisons to work.
func (g *generator) kvValExpr(val ref.Val) string {
	switch val.Type() {
	case types.IntType, types.UintType, types.DoubleType:
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
