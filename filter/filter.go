package filter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

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
}

// Filter is a compiled CEL filter expression.
type Filter struct {
	prg    cel.Program
	celAst *cel.Ast
	env    *cel.Env
	def    *schema.Def
	src    string
}

// Compile parses and type-checks a CEL filter expression.
// When def is non-nil, fields are declared with precise types from the schema.
// When def is nil (schema-free), variables are dynamically typed.
func Compile(expr string, def *schema.Def) (*Filter, error) {
	if expr == "" {
		return nil, fmt.Errorf("%w: empty expression", core.ErrInvalidFilter)
	}

	env, err := buildEnv(def)
	if err != nil {
		return nil, fmt.Errorf("%w: build env: %w", core.ErrInvalidFilter, err)
	}

	// Two-pass compile: first parse to extract identifiers, then extend env
	// with any undeclared variables as DynType. This handles both schema-free
	// mode (all vars dynamic) and schema mode (unknown fields become dynamic
	// so filtering on missing attributes evaluates to false at runtime).
	env, err = extendEnvFromExpr(env, def, expr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", core.ErrInvalidFilter, err)
	}

	celAst, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("%w: compile: %w", core.ErrInvalidFilter, iss.Err())
	}

	prg, err := env.Program(celAst)
	if err != nil {
		return nil, fmt.Errorf("%w: program: %w", core.ErrInvalidFilter, err)
	}

	return &Filter{
		celAst: celAst,
		prg:    prg,
		env:    env,
		src:    expr,
		def:    def,
	}, nil
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
// they type-check in every mode without being schema fields.
func buildEnv(def *schema.Def) (*cel.Env, error) {
	fieldCount := 0
	if def != nil {
		fieldCount = len(def.Fields)
	}
	opts := make([]cel.EnvOption, 0, len(reservedAttrTypes)+fieldCount)

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

// extendEnvFromExpr parses expr to find identifiers, then extends the env
// with any undeclared variables as [cel.DynType]. When def is nil, all
// identifiers are added as dynamic. When def is provided, only identifiers
// not already declared in the schema (or reserved) are added.
//
// Under [schema.ModeStrict], an undeclared, non-reserved identifier is
// rejected instead of being added as dynamic — strict schemas do not
// tolerate filtering on fields they do not define.
func extendEnvFromExpr(env *cel.Env, def *schema.Def, expr string) (*cel.Env, error) {
	celAst, iss := env.Parse(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("parse: %w", iss.Err())
	}

	idents := collectIdents(celAst.NativeRep().Expr())
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
