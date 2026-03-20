package filter

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// Filter is a compiled CEL filter expression.
type Filter struct {
	celAst *cel.Ast
	prg    cel.Program
	env    *cel.Env
	src    string
}

// Compile parses and type-checks a CEL filter expression.
// When def is non-nil, fields are declared with precise types from the schema.
// When def is nil (flexible mode), variables are dynamically typed.
func Compile(expr string, def *schema.Def) (*Filter, error) {
	if expr == "" {
		return nil, fmt.Errorf("filter: empty expression")
	}

	env, err := buildEnv(def)
	if err != nil {
		return nil, fmt.Errorf("filter: build env: %w", err)
	}

	// Two-pass compile: first parse to extract identifiers, then extend env
	// with any undeclared variables as DynType. This handles both schema-free
	// mode (all vars dynamic) and schema mode (unknown fields become dynamic
	// so filtering on missing attributes evaluates to false at runtime).
	env, err = extendEnvFromExpr(env, def, expr)
	if err != nil {
		return nil, fmt.Errorf("filter: %w", err)
	}

	celAst, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("filter: compile: %w", iss.Err())
	}

	prg, err := env.Program(celAst)
	if err != nil {
		return nil, fmt.Errorf("filter: program: %w", err)
	}

	return &Filter{
		celAst: celAst,
		prg:    prg,
		env:    env,
		src:    expr,
	}, nil
}

// Source returns the original expression string.
func (f *Filter) Source() string { return f.src }

// CelAst returns the compiled CEL AST for use by code generators (e.g., SQL).
func (f *Filter) CelAst() *cel.Ast { return f.celAst }

// buildEnv creates a CEL environment from a schema definition.
func buildEnv(def *schema.Def) (*cel.Env, error) {
	if def == nil || len(def.Fields) == 0 {
		return cel.NewEnv()
	}

	opts := make([]cel.EnvOption, 0, len(def.Fields))
	for name, fd := range def.Fields {
		ct := celType(fd.Type)
		opts = append(opts, cel.Variable(name, ct))
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
// not already declared in the schema are added.
func extendEnvFromExpr(env *cel.Env, def *schema.Def, expr string) (*cel.Env, error) {
	celAst, iss := env.Parse(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("parse: %w", iss.Err())
	}

	idents := collectIdents(celAst.NativeRep().Expr())
	if len(idents) == 0 {
		return env, nil
	}

	// Filter out idents already declared in the schema.
	declared := make(map[string]bool)
	if def != nil {
		for name := range def.Fields {
			declared[name] = true
		}
	}

	var extras []cel.EnvOption
	for _, name := range idents {
		if !declared[name] {
			extras = append(extras, cel.Variable(name, cel.DynType))
		}
	}

	if len(extras) == 0 {
		return env, nil
	}

	return env.Extend(extras...)
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
