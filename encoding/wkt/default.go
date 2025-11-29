package wkt

import (
	"fmt"
	"reflect"
	"time"
)

// DefaultRegistry is pre-configured with common well-known types.
// It includes handlers for:
//   - time.Time (marshaled as RFC3339 strings)
//   - time.Duration (marshaled as nanosecond int64 values)
//
// This registry is safe for concurrent use and can be used directly
// or cloned and customized.
var DefaultRegistry *Registry

func init() {
	DefaultRegistry = NewRegistry()
	registerBuiltinTypes(DefaultRegistry)
}

// registerBuiltinTypes registers the built-in well-known types.
func registerBuiltinTypes(r *Registry) {
	registerTime(r)
	registerDuration(r)
}

// registerTime registers time.Time with RFC3339 encoding.
func registerTime(r *Registry) {
	r.Register(
		reflect.TypeOf(time.Time{}),
		func(v reflect.Value) (any, error) {
			t := v.Interface().(time.Time)
			if t.IsZero() {
				return "", nil
			}
			return t.Format(time.RFC3339Nano), nil
		},
		func(v any, target reflect.Value) error {
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf("wkt: expected string for time.Time, got %T", v)
			}
			if str == "" {
				target.Set(reflect.ValueOf(time.Time{}))
				return nil
			}
			t, err := time.Parse(time.RFC3339Nano, str)
			if err != nil {
				return fmt.Errorf("wkt: failed to parse time: %w", err)
			}
			target.Set(reflect.ValueOf(t))
			return nil
		},
	)
}

// registerDuration registers time.Duration as int64 nanoseconds.
func registerDuration(r *Registry) {
	r.Register(
		reflect.TypeOf(time.Duration(0)),
		func(v reflect.Value) (any, error) {
			d := v.Interface().(time.Duration)
			return int64(d), nil
		},
		func(v any, target reflect.Value) error {
			var nanos int64
			switch val := v.(type) {
			case int64:
				nanos = val
			case int:
				nanos = int64(val)
			case float64:
				nanos = int64(val)
			default:
				return fmt.Errorf("wkt: expected numeric value for time.Duration, got %T", v)
			}
			target.Set(reflect.ValueOf(time.Duration(nanos)))
			return nil
		},
	)
}
