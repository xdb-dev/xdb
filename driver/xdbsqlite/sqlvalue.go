package xdbsqlite

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gojekfarm/xtools/errors"
	"github.com/spf13/cast"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

type sqlValue struct {
	attr  *types.Attribute
	value *types.Value
}

// Value returns a SQLite compatible type for a xdb/types.Value.
// mapping:
// - string -> TEXT
// - int -> INTEGER
// - float -> REAL
// - bool -> INTEGER
// - bytes -> BLOB
// - time -> INTEGER
// - []string -> TEXT(JSON)
// - []int -> TEXT(JSON)
// - []float -> TEXT(JSON)
// - []bool -> TEXT(JSON)
// - []bytes -> TEXT(JSON)
// - []time -> TEXT(JSON)
func (s *sqlValue) Value() (any, error) {
	switch s.value.Type().ID() {
	case types.TypeIDBoolean:
		if s.value.ToBool() {
			return 1, nil
		}
		return 0, nil
	case types.TypeIDInteger:
		return s.value.ToInt(), nil
	case types.TypeIDUnsigned:
		return s.value.ToUint(), nil
	case types.TypeIDFloat:
		return s.value.ToFloat(), nil
	case types.TypeIDString:
		return s.value.ToString(), nil
	case types.TypeIDBytes:
		return s.value.ToBytes(), nil
	case types.TypeIDTime:
		return s.value.ToTime().UnixMilli(), nil
	case types.TypeIDArray:
		switch s.value.Type().ValueType() {
		case types.TypeIDInteger, types.TypeIDUnsigned:
			// Convert integers to strings to maintain precision
			values := x.Map(s.value.ToIntArray(), func(v int64) string {
				return fmt.Sprintf("%d", v)
			})
			b, err := json.Marshal(values)
			if err != nil {
				return nil, err
			}
			return string(b), nil
		case types.TypeIDTime:
			values := x.Map(s.value.ToTimeArray(), func(v time.Time) string {
				return fmt.Sprintf("%d", v.UnixMilli())
			})
			b, err := json.Marshal(values)
			if err != nil {
				return nil, err
			}
			return string(b), nil
		default:
			b, err := json.Marshal(s.value.Unwrap())
			if err != nil {
				return nil, err
			}
			return string(b), nil
		}
	default:
		return nil, errors.Wrap(ErrUnsupportedValue, "type", s.value.Type().Name())
	}
}

// Scan converts a SQLite compatible type to a xdb/types.Value.
func (s *sqlValue) Scan(src any) error {
	if src == nil {
		return nil
	}
	switch s.attr.Type.ID() {
	case types.TypeIDArray:
		var decoded []any
		if err := json.Unmarshal([]byte(src.(string)), &decoded); err != nil {
			return err
		}
		val, err := castValue(decoded, s.attr.Type.ValueType())
		if err != nil {
			return err
		}
		s.value = val
	default:
		val, err := castValue(src, s.attr.Type.ID())
		if err != nil {
			return err
		}
		s.value = val
	}
	return nil
}

func castValue(src any, typ types.TypeID) (*types.Value, error) {
	switch typ {
	case types.TypeIDBoolean:
		return types.NewSafeValue(cast.ToBool(src))
	case types.TypeIDInteger:
		return types.NewSafeValue(cast.ToInt64(src))
	case types.TypeIDUnsigned:
		return types.NewSafeValue(cast.ToUint64(src))
	case types.TypeIDFloat:
		return types.NewSafeValue(cast.ToFloat64(src))
	case types.TypeIDString:
		return types.NewSafeValue(cast.ToString(src))
	case types.TypeIDBytes:
		if str, ok := src.(string); ok {
			b64, err := base64.StdEncoding.DecodeString(str)
			if err != nil {
				return nil, err
			}
			return types.NewSafeValue(b64)
		}
		return types.NewSafeValue(src.([]byte))
	case types.TypeIDTime:
		return types.NewSafeValue(time.UnixMilli(cast.ToInt64(src)))
	case types.TypeIDArray:
		values := x.Map(src.([]any), func(v any) *types.Value {
			val, err := castValue(v, typ)
			if err != nil {
				return nil
			}
			return val
		})
		return types.NewSafeValue(values)
	}
	return nil, nil
}
