package filter

import (
	"fmt"

	"github.com/google/cel-go/common/types"

	"github.com/xdb-dev/xdb/core"
)

// Match evaluates the compiled [Filter] against a [core.Record].
// Returns true if the record satisfies the filter, false otherwise.
// A missing attribute causes comparisons against it to return false.
func (f *Filter) Match(record *core.Record) (bool, error) {
	act := buildActivation(record)

	out, _, evalErr := f.prg.Eval(act)
	if evalErr != nil {
		// CEL returns an error for missing variables or type mismatches.
		// Treat as non-matching rather than propagating.
		return false, nil //nolint:nilerr // intentional: missing vars → no match
	}

	if out.Type() != types.BoolType {
		return false, fmt.Errorf("filter: expression did not evaluate to bool, got %s", out.Type())
	}

	return out.Value().(bool), nil
}

// buildActivation converts a record's tuples into a map[string]any suitable
// for CEL evaluation.
func buildActivation(record *core.Record) map[string]any {
	tuples := record.Tuples()
	act := make(map[string]any, len(tuples))

	for _, tuple := range tuples {
		act[tuple.Attr()] = nativeValue(tuple.Value())
	}

	return act
}

// Filter returns only the records that match the compiled filter.
// This is the primary integration point for in-memory stores.
func (f *Filter) Filter(records []*core.Record) ([]*core.Record, error) {
	var result []*core.Record
	for _, r := range records {
		ok, err := f.Match(r)
		if err != nil {
			return nil, err
		}
		if ok {
			result = append(result, r)
		}
	}
	return result, nil
}

// nativeValue converts a [core.Value] to a native Go type that CEL understands.
func nativeValue(v *core.Value) any {
	if v == nil {
		return nil
	}

	switch v.Type().ID() {
	case core.TIDString:
		s, err := v.AsStr()
		if err != nil {
			return v.String()
		}
		return s

	case core.TIDInteger:
		i, err := v.AsInt()
		if err != nil {
			return v.String()
		}
		return i

	case core.TIDUnsigned:
		u, err := v.AsUint()
		if err != nil {
			return v.String()
		}
		return u

	case core.TIDFloat:
		f, err := v.AsFloat()
		if err != nil {
			return v.String()
		}
		return f

	case core.TIDBoolean:
		b, err := v.AsBool()
		if err != nil {
			return v.String()
		}
		return b

	case core.TIDTime:
		t, err := v.AsTime()
		if err != nil {
			return v.String()
		}
		return t

	default:
		return v.String()
	}
}
