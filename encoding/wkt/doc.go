// Package wkt provides a registry for well-known types in the XDB encoding system.
//
// Well-known types are custom Go types that should be handled specially during
// encoding/decoding rather than being flattened into dot-separated paths. This
// includes common types like time.Time, time.Duration, and UUID, as well as
// user-defined types that need custom marshaling logic.
//
// # Overview
//
// The wkt package allows you to register custom marshal/unmarshal functions for
// specific Go types. When the xdbstruct encoder encounters a registered type, it
// uses the custom marshal function instead of flattening the struct fields.
//
// # Basic Usage
//
//	import "github.com/xdb-dev/xdb/encoding/wkt"
//
//	// Use the default registry with built-in types
//	encoder := xdbstruct.NewEncoder(xdbstruct.Options{
//	    Tag:      "xdb",
//	    Registry: wkt.DefaultRegistry,
//	})
//
// # Custom Type Registration
//
// You can create a custom registry and register your own types:
//
//	type Money struct {
//	    Amount   int64
//	    Currency string
//	}
//
//	registry := wkt.NewRegistry()
//	registry.Register(
//	    reflect.TypeOf(Money{}),
//	    func(v reflect.Value) (any, error) {
//	        m := v.Interface().(Money)
//	        return fmt.Sprintf("%d %s", m.Amount, m.Currency), nil
//	    },
//	    func(v any, target reflect.Value) error {
//	        str := v.(string)
//	        // Parse "10000 USD" back to Money
//	        parts := strings.Split(str, " ")
//	        amount, _ := strconv.ParseInt(parts[0], 10, 64)
//	        money := Money{Amount: amount, Currency: parts[1]}
//	        target.Set(reflect.ValueOf(money))
//	        return nil
//	    },
//	)
//
// # Built-in Types
//
// The DefaultRegistry includes handlers for:
//   - time.Time
//   - time.Duration
//   - uuid.UUID (if github.com/google/uuid is available)
//
// # Type Processing Priority
//
// When the xdbstruct encoder encounters a field, it processes it in this order:
//  1. Check WKT registry - If registered, use custom marshal function
//  2. Check for marshalers - json.Marshaler or encoding.BinaryMarshaler
//  3. Default: flatten - Nested struct fields become dot-separated paths
package wkt
