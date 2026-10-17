package filter

import (
	"fmt"
	"slices"
	"strings"
	"time"

	xerrors "github.com/gojekfarm/xtools/errors"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// GrammarHint is the one-line pointer attached to every invalid-filter
// error as its fix tag. It reaches the CLI as the hint of the error
// envelope. An embedder with no xdb CLI can find it in the error text and
// replace it with its own pointer.
const GrammarHint = "run xdb describe --filter to see the filter grammar"

// NowVar is the filter variable that holds the time of the evaluation, for
// a filter such as start <= now + duration("15m"). A schema field of the
// same name wins, as a field wins over a reserved attribute: a filter on a
// declared field must keep its meaning.
const NowVar = "now"

// AttrsVar is the reserved filter variable that holds the attribute names
// a record carries. Write "name" in _attrs, or has(name), to select the
// records that hold an attribute.
const AttrsVar = "_attrs"

// reservedAttrTypes maps reserved attribute names to their CEL type. These
// attributes are always filterable, in every schema mode, without being
// declared as schema fields — they never trip strict mode's unknown-field
// rejection.
//
// [schema.FieldVersion] and [schema.FieldUpdated] are stamped onto every
// definition, so a stored def usually declares them and wins here.
// [schema.FieldID] never is: it is projected from the record path rather
// than stored, so this is the only place it is declared.
var reservedAttrTypes = map[string]*cel.Type{
	schema.FieldID:      cel.StringType,
	schema.FieldVersion: cel.IntType,
	schema.FieldUpdated: cel.TimestampType,
	AttrsVar:            cel.ListType(cel.StringType),
}

// presenceMacro replaces the standard has() macro. An XDB attribute name is
// flat, and a dotted name such as author.name is one attribute, not a field
// of a nested message. The standard macro rejects a bare identifier and
// reads a dotted name as a field selection, so neither form asks the
// question a caller means. Both expand to a membership test on [AttrsVar].
var presenceMacro = cel.GlobalMacro("has", 1, expandPresence)

// expandPresence rewrites has(attr) to "attr" in _attrs.
func expandPresence(
	mef cel.MacroExprFactory,
	_ ast.Expr,
	args []ast.Expr,
) (ast.Expr, *cel.Error) {
	name, ok := attrPath(args[0])
	if !ok {
		return nil, mef.NewError(args[0].ID(),
			"has() expects an attribute name, for example has(assignee)")
	}

	return mef.NewCall(operators.In,
		mef.NewLiteral(types.String(name)),
		mef.NewIdent(AttrsVar),
	), nil
}

// attrPath renders an identifier or a chain of field selections as one
// dotted attribute name. Any other expression has no attribute name.
func attrPath(expr ast.Expr) (string, bool) {
	switch expr.Kind() {
	case ast.IdentKind:
		return expr.AsIdent(), true

	case ast.SelectKind:
		sel := expr.AsSelect()
		if sel.IsTestOnly() {
			return "", false
		}

		parent, ok := attrPath(sel.Operand())
		if !ok {
			return "", false
		}

		return parent + "." + sel.FieldName(), true

	default:
		return "", false
	}
}

// Filter is a compiled CEL filter expression.
type Filter struct {
	prg    cel.Program
	celAst *cel.Ast
	env    *cel.Env
	def    *schema.Def
	src    string
	clock  bool // NowVar is the time of the evaluation, not a field
}

// Compile parses and type-checks a CEL filter expression.
// When def is non-nil, fields are declared with precise types from the schema.
// When def is nil (schema-free), variables are dynamically typed.
func Compile(expr string, def *schema.Def) (*Filter, error) {
	if expr == "" {
		return nil, invalidFilter(fmt.Errorf("empty expression"))
	}

	clock := isClock(def)

	env, err := buildEnv(def, clock)
	if err != nil {
		return nil, invalidFilter(fmt.Errorf("build env: %w", err))
	}

	parsed, iss := env.Parse(expr)
	if iss.Err() != nil {
		return nil, invalidFilter(fmt.Errorf("parse: %w", iss.Err()))
	}

	// Two-pass compile: first parse to extract identifiers, then extend env
	// with any undeclared variables as DynType. This handles both schema-free
	// mode (all vars dynamic) and schema mode (unknown fields become dynamic
	// so filtering on missing attributes evaluates to false at runtime).
	env, err = extendEnvFromExpr(env, def, clock, parsed)
	if err != nil {
		return nil, invalidFilter(err)
	}

	if err := coerceTimeStrings(parsed, def, clock); err != nil {
		return nil, invalidFilter(err)
	}

	celAst, iss := env.Check(parsed)
	if iss.Err() != nil {
		return nil, invalidFilter(fmt.Errorf("compile: %w", iss.Err()))
	}

	if validErr := validateExpr(celAst.NativeRep().Expr(), def); validErr != nil {
		return nil, invalidFilter(validErr)
	}

	prg, err := env.Program(celAst)
	if err != nil {
		return nil, invalidFilter(fmt.Errorf("program: %w", err))
	}

	return &Filter{
		celAst: celAst,
		prg:    prg,
		env:    env,
		src:    expr,
		def:    def,
		clock:  clock,
	}, nil
}

// Clock reports whether [NowVar] is the time of the evaluation, not a
// declared field. A SQL generator must not render the clock as a column.
func (f *Filter) Clock() bool {
	return f.clock
}

// isClock reports whether [NowVar] is the time of the evaluation. It is,
// unless def declares a field of that name.
func isClock(def *schema.Def) bool {
	if def == nil {
		return true
	}

	_, declared := def.Fields[NowVar]

	return !declared
}

// comparisons are the CEL operators that compare two values.
var comparisons = map[string]bool{
	operators.Equals:        true,
	operators.NotEquals:     true,
	operators.Less:          true,
	operators.LessEquals:    true,
	operators.Greater:       true,
	operators.GreaterEquals: true,
}

// coerceTimeStrings rewrites a string literal that is compared with a time
// field into a timestamp() call, so start >= "2026-09-29T00:00:00Z" reads
// as it does with timestamp(). It rewrites only against a field whose time
// type is known before evaluation: a declared time field, _updated, or the
// clock. A dynamically typed field can hold a string, so its comparison
// stays a string comparison. A literal that is not RFC 3339 is an error.
func coerceTimeStrings(parsed *cel.Ast, def *schema.Def, clock bool) error {
	native := parsed.NativeRep()
	nextID := ast.MaxID(native)
	factory := ast.NewExprFactory()

	var coerceErr error

	visit := func(expr ast.Expr) {
		if coerceErr != nil || expr.Kind() != ast.CallKind {
			return
		}

		call := expr.AsCall()
		args := call.Args()
		if !comparisons[call.FunctionName()] || len(args) != 2 {
			return
		}

		for i, arg := range args {
			other := args[1-i]

			text, isString := stringLiteral(arg)
			if !isString || !isTimeOperand(other, def, clock) {
				continue
			}

			if _, err := time.Parse(time.RFC3339, text); err != nil {
				coerceErr = fmt.Errorf(
					"%q is not an RFC 3339 time, for example \"2026-08-01T00:00:00Z\"",
					text,
				)

				return
			}

			literal := factory.NewLiteral(nextID, types.String(text))
			timestamp := factory.NewCall(nextID+1, "timestamp", literal)
			nextID += 2

			arg.SetKindCase(timestamp)
		}
	}

	ast.PostOrderVisit(native.Expr(), ast.NewExprVisitor(visit))

	return coerceErr
}

// stringLiteral returns the value of a string literal.
func stringLiteral(expr ast.Expr) (string, bool) {
	if expr.Kind() != ast.LiteralKind {
		return "", false
	}

	text, ok := expr.AsLiteral().Value().(string)

	return text, ok
}

// isTimeOperand reports whether expr names a value whose type is a time
// before evaluation.
func isTimeOperand(expr ast.Expr, def *schema.Def, clock bool) bool {
	if expr.Kind() != ast.IdentKind {
		return false
	}

	name := expr.AsIdent()

	if def != nil {
		if field, ok := def.Fields[name]; ok {
			return field.Type.ID() == core.TIDTime
		}
	}

	if name == NowVar {
		return clock
	}

	return reservedAttrTypes[name] == cel.TimestampType
}

// invalidFilter wraps err as [core.ErrInvalidFilter] and attaches the fix
// tag. Every rejection from [Compile] goes through it, so a caller always
// gets the same sentinel and the same pointer to the grammar.
func invalidFilter(err error) error {
	return xerrors.Wrap(
		fmt.Errorf("%w: %w", core.ErrInvalidFilter, err),
		"fix", GrammarHint,
		"reason", "invalid_filter",
	)
}

// validateExpr walks a compiled AST for faults that CEL accepts but XDB
// rejects: a constant timestamp() that is not an RFC 3339 time, and a
// presence test on a field that a strict schema does not declare. CEL
// defers both to evaluation, where a filter error reads as "no records
// match". A filter that can never match any record is a caller mistake, so
// it is reported at compile time instead.
func validateExpr(expr ast.Expr, def *schema.Def) error {
	if expr == nil {
		return nil
	}

	switch expr.Kind() {
	case ast.CallKind:
		call := expr.AsCall()
		args := call.Args()

		if call.FunctionName() == "timestamp" && len(args) == 1 {
			if err := checkTimestampArg(args[0]); err != nil {
				return err
			}
		}

		if name, ok := presenceAttr(call); ok {
			if err := checkPresenceAttr(name, def); err != nil {
				return err
			}
		}

		if call.IsMemberFunction() {
			if err := validateExpr(call.Target(), def); err != nil {
				return err
			}
		}

		for _, arg := range args {
			if err := validateExpr(arg, def); err != nil {
				return err
			}
		}

	case ast.SelectKind:
		return validateExpr(expr.AsSelect().Operand(), def)

	case ast.ListKind:
		for _, elem := range expr.AsList().Elements() {
			if err := validateExpr(elem, def); err != nil {
				return err
			}
		}
	}

	return nil
}

// presenceAttr returns the attribute name of an expanded has() call, which
// is a membership test of a constant name against [AttrsVar].
func presenceAttr(call ast.CallExpr) (string, bool) {
	args := call.Args()
	if call.FunctionName() != operators.In || len(args) != 2 {
		return "", false
	}

	if args[1].Kind() != ast.IdentKind || args[1].AsIdent() != AttrsVar {
		return "", false
	}

	if args[0].Kind() != ast.LiteralKind {
		return "", false
	}

	name, ok := args[0].AsLiteral().Value().(string)

	return name, ok
}

// checkPresenceAttr rejects a presence test on a field that a strict schema
// does not declare. The macro turns the name into a string literal, so the
// unknown-identifier check in extendEnvFromExpr never sees it.
func checkPresenceAttr(name string, def *schema.Def) error {
	if def == nil || def.Mode != schema.ModeStrict {
		return nil
	}

	if _, ok := def.Fields[name]; ok {
		return nil
	}

	if _, ok := reservedAttrTypes[name]; ok {
		return nil
	}

	return unknownFieldError(name, def)
}

// checkTimestampArg parses a constant timestamp() argument. A non-constant
// argument passes: only the record supplies its value.
func checkTimestampArg(arg ast.Expr) error {
	if arg.Kind() != ast.LiteralKind {
		return nil
	}

	text, ok := arg.AsLiteral().Value().(string)
	if !ok {
		return nil
	}

	if _, err := time.Parse(time.RFC3339, text); err != nil {
		return fmt.Errorf(
			"timestamp(%q) is not an RFC 3339 time, for example timestamp(\"2026-08-01T00:00:00Z\")",
			text,
		)
	}

	return nil
}

// Source returns the original expression string.
func (f *Filter) Source() string { return f.src }

// CelAst returns the compiled CEL AST for use by code generators (e.g., SQL).
func (f *Filter) CelAst() *cel.Ast { return f.celAst }

// Def returns the schema definition this filter was compiled against, or
// nil when compiled without one (schema-free).
func (f *Filter) Def() *schema.Def { return f.def }

// buildEnv creates a CEL environment from a schema definition. Reserved
// attributes (see reservedAttrTypes) are declared regardless of def, so
// they type-check in every mode without being schema fields. The strings
// extension adds the case functions, such as lowerAscii(). When clock is
// set, [NowVar] is declared as a timestamp.
func buildEnv(def *schema.Def, clock bool) (*cel.Env, error) {
	fieldCount := 0
	if def != nil {
		fieldCount = len(def.Fields)
	}
	opts := make([]cel.EnvOption, 0, len(reservedAttrTypes)+fieldCount+3)
	opts = append(opts,
		cel.Macros(presenceMacro),
		ext.Strings(),
	)

	if clock {
		opts = append(opts, cel.Variable(NowVar, cel.TimestampType))
	}

	for name, ct := range reservedAttrTypes {
		if def != nil {
			if _, ok := def.Fields[name]; ok {
				continue // schema field wins over the reserved default.
			}
		}
		opts = append(opts, cel.Variable(name, ct))
	}

	if def != nil {
		for name, fd := range def.Fields {
			opts = append(opts, cel.Variable(name, celType(fd.Type.ID())))
		}
	}

	return cel.NewEnv(opts...)
}

// celType maps a [core.TID] to a CEL type.
func celType(tid core.TID) *cel.Type {
	switch tid {
	case core.TIDString:
		return cel.StringType
	case core.TIDInteger:
		return cel.IntType
	case core.TIDUnsigned:
		return cel.UintType
	case core.TIDFloat:
		return cel.DoubleType
	case core.TIDBoolean:
		return cel.BoolType
	case core.TIDTime:
		return cel.TimestampType
	default:
		return cel.DynType
	}
}

// extendEnvFromExpr walks a parsed expression to find identifiers, then extends the env
// with any undeclared variables as [cel.DynType]. When def is nil, all
// identifiers are added as dynamic. When def is provided, only identifiers
// not already declared in the schema (or reserved) are added.
//
// Under [schema.ModeStrict], an undeclared, non-reserved identifier is
// rejected instead of being added as dynamic — strict schemas do not
// tolerate filtering on fields they do not define.
func extendEnvFromExpr(
	env *cel.Env,
	def *schema.Def,
	clock bool,
	parsed *cel.Ast,
) (*cel.Env, error) {
	idents := collectIdents(parsed.NativeRep().Expr())
	if len(idents) == 0 {
		return env, nil
	}

	// Filter out idents already declared in the schema or reserved.
	declared := make(map[string]bool)
	if def != nil {
		for name := range def.Fields {
			declared[name] = true
		}
	}
	for name := range reservedAttrTypes {
		declared[name] = true
	}
	if clock {
		declared[NowVar] = true
	}

	extras := make([]cel.EnvOption, 0, len(idents))
	for _, name := range idents {
		if declared[name] {
			continue
		}
		if def != nil && def.Mode == schema.ModeStrict {
			return nil, unknownFieldError(name, def)
		}
		extras = append(extras, cel.Variable(name, cel.DynType))
	}

	if len(extras) == 0 {
		return env, nil
	}

	return env.Extend(extras...)
}

// unknownFieldError reports a filter identifier that a strict schema does
// not declare, naming the field and listing the schema's available fields
// in sorted order.
func unknownFieldError(name string, def *schema.Def) error {
	names := make([]string, 0, len(def.Fields))
	for n := range def.Fields {
		names = append(names, n)
	}
	slices.Sort(names)

	return fmt.Errorf("unknown field %q in filter; available fields: %s",
		name, strings.Join(names, ", "))
}

// collectIdents recursively extracts unique identifier names from a CEL AST.
func collectIdents(expr ast.Expr) []string {
	seen := make(map[string]bool)
	walkIdents(expr, seen)

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}

// walkIdents recursively walks an AST node and collects identifier names.
func walkIdents(expr ast.Expr, seen map[string]bool) {
	if expr == nil {
		return
	}

	switch expr.Kind() {
	case ast.IdentKind:
		seen[expr.AsIdent()] = true

	case ast.CallKind:
		call := expr.AsCall()
		if call.IsMemberFunction() {
			walkIdents(call.Target(), seen)
		}
		for _, arg := range call.Args() {
			walkIdents(arg, seen)
		}

	case ast.SelectKind:
		sel := expr.AsSelect()
		walkIdents(sel.Operand(), seen)

	case ast.ListKind:
		list := expr.AsList()
		for _, elem := range list.Elements() {
			walkIdents(elem, seen)
		}
	}
}
