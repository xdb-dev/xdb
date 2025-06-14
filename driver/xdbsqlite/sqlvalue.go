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
	value types.Value
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
	switch v := s.value.(type) {
	case types.Bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case types.Int64:
		return int64(v), nil
	case types.Uint64:
		return uint64(v), nil
	case types.Float64:
		return float64(v), nil
	case types.String:
		return string(v), nil
	case types.Bytes:
		return v, nil
	case types.Time:
		return time.Time(v).UnixMilli(), nil
	case *types.Array:
		switch v.ValueType().ID() {
		case types.TypeInteger, types.TypeUnsigned:
			// Convert integers to strings to maintain precision
			values := x.Map(v.Values(), func(v types.Value) string {
				return fmt.Sprintf("%d", v)
			})
			b, err := json.Marshal(values)
			if err != nil {
				return nil, err
			}
			return string(b), nil
		case types.TypeTime:
			values := x.Map(v.Values(), func(v types.Value) string {
				return fmt.Sprintf("%d", time.Time(v.(types.Time)).UnixMilli())
			})
			b, err := json.Marshal(values)
			if err != nil {
				return nil, err
			}
			return string(b), nil
		default:
			b, err := json.Marshal(v.Values())
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
	case types.TypeArray:
		var decoded []any
		if err := json.Unmarshal([]byte(src.(string)), &decoded); err != nil {
			return err
		}
		s.value = castValue(decoded, s.attr.Type)
	default:
		s.value = castValue(src, s.attr.Type)
	}
	return nil
}

func castValue(src any, typ types.Type) types.Value {
	switch typ.ID() {
	case types.TypeBoolean:
		return types.Bool(cast.ToBool(src))
	case types.TypeInteger:
		return types.Int64(cast.ToInt64(src))
	case types.TypeUnsigned:
		return types.Uint64(cast.ToUint64(src))
	case types.TypeFloat:
		return types.Float64(cast.ToFloat64(src))
	case types.TypeString:
		return types.String(cast.ToString(src))
	case types.TypeBytes:
		if str, ok := src.(string); ok {
			b64, err := base64.StdEncoding.DecodeString(str)
			if err != nil {
				return nil
			}
			return types.Bytes(b64)
		}
		return types.Bytes(src.([]byte))
	case types.TypeTime:
		return types.Time(time.UnixMilli(cast.ToInt64(src)))
	case types.TypeArray:
		at := typ.(types.ArrayType)
		values := x.Map(src.([]any), func(v any) types.Value {
			return castValue(v, at.ValueType())
		})
		return types.NewArray(at.ValueType(), values...)
	}
	return nil
}
