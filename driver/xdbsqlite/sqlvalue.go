package xdbsqlite

// import (
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"github.com/gojekfarm/xtools/errors"
// 	"github.com/spf13/cast"

// 	"github.com/xdb-dev/xdb/core"
// 	"github.com/xdb-dev/xdb/x"
// )

// type sqlValue struct {
// 	attr  *core.Attribute
// 	value *core.Value
// }

// // Value returns a SQLite compatible type for a xdb/core.Value.
// // mapping:
// // - string -> TEXT
// // - int -> INTEGER
// // - float -> REAL
// // - bool -> INTEGER
// // - bytes -> BLOB
// // - time -> INTEGER
// // - []string -> TEXT(JSON)
// // - []int -> TEXT(JSON)
// // - []float -> TEXT(JSON)
// // - []bool -> TEXT(JSON)
// // - []bytes -> TEXT(JSON)
// // - []time -> TEXT(JSON)
// func (s *sqlValue) Value() (any, error) {
// 	switch s.value.Type().ID() {
// 	case core.TypeIDBoolean:
// 		if s.value.ToBool() {
// 			return 1, nil
// 		}
// 		return 0, nil
// 	case core.TypeIDInteger:
// 		return s.value.ToInt(), nil
// 	case core.TypeIDUnsigned:
// 		return s.value.ToUint(), nil
// 	case core.TypeIDFloat:
// 		return s.value.ToFloat(), nil
// 	case core.TypeIDString:
// 		return s.value.ToString(), nil
// 	case core.TypeIDBytes:
// 		return s.value.ToBytes(), nil
// 	case core.TypeIDTime:
// 		return s.value.ToTime().UnixMilli(), nil
// 	case core.TypeIDArray:
// 		switch s.value.Type().ValueType() {
// 		case core.TypeIDInteger, core.TypeIDUnsigned:
// 			// Convert integers to strings to maintain precision
// 			values := x.Map(s.value.ToIntArray(), func(v int64) string {
// 				return fmt.Sprintf("%d", v)
// 			})
// 			b, err := json.Marshal(values)
// 			if err != nil {
// 				return nil, err
// 			}
// 			return string(b), nil
// 		case core.TypeIDTime:
// 			values := x.Map(s.value.ToTimeArray(), func(v time.Time) string {
// 				return fmt.Sprintf("%d", v.UnixMilli())
// 			})
// 			b, err := json.Marshal(values)
// 			if err != nil {
// 				return nil, err
// 			}
// 			return string(b), nil
// 		default:
// 			b, err := json.Marshal(s.value.Unwrap())
// 			if err != nil {
// 				return nil, err
// 			}
// 			return string(b), nil
// 		}
// 	default:
// 		return nil, errors.Wrap(ErrUnsupportedValue, "type", s.value.Type().Name())
// 	}
// }

// // Scan converts a SQLite compatible type to a xdb/core.Value.
// func (s *sqlValue) Scan(src any) error {
// 	if src == nil {
// 		return nil
// 	}
// 	switch s.attr.Type.ID() {
// 	case core.TypeIDArray:
// 		var decoded []any
// 		if err := json.Unmarshal([]byte(src.(string)), &decoded); err != nil {
// 			return err
// 		}
// 		val, err := castValue(decoded, s.attr.Type.ValueType())
// 		if err != nil {
// 			return err
// 		}
// 		s.value = val
// 	default:
// 		val, err := castValue(src, s.attr.Type.ID())
// 		if err != nil {
// 			return err
// 		}
// 		s.value = val
// 	}
// 	return nil
// }

// func castValue(src any, typ core.TypeID) (*core.Value, error) {
// 	switch typ {
// 	case core.TypeIDBoolean:
// 		return core.NewSafeValue(cast.ToBool(src))
// 	case core.TypeIDInteger:
// 		return core.NewSafeValue(cast.ToInt64(src))
// 	case core.TypeIDUnsigned:
// 		return core.NewSafeValue(cast.ToUint64(src))
// 	case core.TypeIDFloat:
// 		return core.NewSafeValue(cast.ToFloat64(src))
// 	case core.TypeIDString:
// 		return core.NewSafeValue(cast.ToString(src))
// 	case core.TypeIDBytes:
// 		if str, ok := src.(string); ok {
// 			b64, err := base64.StdEncoding.DecodeString(str)
// 			if err != nil {
// 				return nil, err
// 			}
// 			return core.NewSafeValue(b64)
// 		}
// 		return core.NewSafeValue(src.([]byte))
// 	case core.TypeIDTime:
// 		return core.NewSafeValue(time.UnixMilli(cast.ToInt64(src)))
// 	case core.TypeIDArray:
// 		values := x.Map(src.([]any), func(v any) *core.Value {
// 			val, err := castValue(v, typ)
// 			if err != nil {
// 				return nil
// 			}
// 			return val
// 		})
// 		return core.NewSafeValue(values)
// 	}
// 	return nil, nil
// }
