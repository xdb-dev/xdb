package xdbsqlite

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"github.com/gojekfarm/xtools/errors"
	"github.com/spf13/cast"

	"github.com/xdb-dev/xdb/codec/json"
	"github.com/xdb-dev/xdb/core"
)

var (
	ErrValueConversion = errors.New("[xdbsqlite] value conversion error")
	ErrNilFieldType    = errors.New("[xdbsqlite] nil field type")
)

var (
	_ driver.Valuer = (*value)(nil)
	_ sql.Scanner   = (*value)(nil)
)

var codec = json.New()

// value bridges core.Value and SQL by implementing driver.Valuer and sql.Scanner.
type value struct {
	val *core.Value
	typ core.Type
}

func newValue(val *core.Value) *value {
	return &value{val: val, typ: val.Type()}
}

func newValueFor(typ core.Type) *value {
	return &value{typ: typ}
}

func (v *value) Unwrap() *core.Value {
	return v.val
}

func (v *value) Type() core.Type {
	return v.typ
}

func (v *value) Value() (driver.Value, error) {
	if v.val == nil || v.val.IsNil() {
		return nil, nil
	}

	switch v.val.Type().ID() {
	case core.TIDString:
		return v.val.ToString(), nil
	case core.TIDInteger:
		return v.val.ToInt(), nil
	case core.TIDUnsigned:
		return v.val.ToInt(), nil
	case core.TIDBoolean:
		if v.val.ToBool() {
			return int64(1), nil
		}
		return int64(0), nil
	case core.TIDFloat:
		return v.val.ToFloat(), nil
	case core.TIDBytes:
		return v.val.ToBytes(), nil
	case core.TIDTime:
		return v.val.ToTime().UnixMilli(), nil
	case core.TIDArray, core.TIDMap:
		b, err := codec.EncodeValue(v.val)
		if err != nil {
			return nil, err
		}
		return string(b), nil
	default:
		return nil, errors.Wrap(ErrValueConversion, "type", v.val.Type().ID().String())
	}
}

// Scan implements the sql.Scanner interface.
func (v *value) Scan(src any) error {
	if src == nil {
		v.val = nil
		return nil
	}

	val, err := scanValue(src, v.typ)
	if err != nil {
		return err
	}

	v.val = val
	return nil
}

func scanValue(src any, typ core.Type) (*core.Value, error) {
	switch typ.ID() {
	case core.TIDString:
		s, err := cast.ToStringE(src)
		if err != nil {
			return nil, err
		}
		return core.NewValue(s), nil
	case core.TIDInteger:
		return scanInt64(src)
	case core.TIDUnsigned:
		return scanUint64(src)
	case core.TIDBoolean:
		b, err := cast.ToBoolE(src)
		if err != nil {
			return nil, err
		}
		return core.NewValue(b), nil
	case core.TIDFloat:
		f, err := cast.ToFloat64E(src)
		if err != nil {
			return nil, err
		}
		return core.NewValue(f), nil
	case core.TIDBytes:
		b, err := cast.ToStringE(src)
		if err != nil {
			return nil, err
		}
		return core.NewValue([]byte(b)), nil
	case core.TIDTime:
		return scanTime(src)
	case core.TIDArray, core.TIDMap:
		s, err := cast.ToStringE(src)
		if err != nil {
			return nil, err
		}
		return codec.DecodeValue([]byte(s))
	default:
		return nil, errors.Wrap(ErrValueConversion, "type", typ.ID().String())
	}
}

func scanInt64(src any) (*core.Value, error) {
	switch n := src.(type) {
	case int64:
		return core.NewValue(n), nil
	case float64:
		return core.NewValue(int64(n)), nil
	default:
		return nil, errors.Wrap(ErrValueConversion, "type", "integer")
	}
}

func scanUint64(src any) (*core.Value, error) {
	switch n := src.(type) {
	case int64:
		return core.NewValue(uint64(n)), nil
	case float64:
		return core.NewValue(uint64(n)), nil
	default:
		return nil, errors.Wrap(ErrValueConversion, "type", "unsigned")
	}
}

func scanTime(src any) (*core.Value, error) {
	switch n := src.(type) {
	case int64:
		return core.NewValue(time.UnixMilli(n).UTC()), nil
	case float64:
		return core.NewValue(time.UnixMilli(int64(n)).UTC()), nil
	default:
		return nil, errors.Wrap(ErrValueConversion, "type", "time")
	}
}
