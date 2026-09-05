package sql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Value pairs a column name with its [core.Type] and [*core.Value].
// It implements [driver.Valuer] for writes and [sql.Scanner] for reads.
type Value struct {
	Val  *core.Value
	Type core.Type
	Name string
}

// Value implements [driver.Valuer]. database/sql calls this automatically
// when a Value is passed as a query argument.
func (v Value) Value() (driver.Value, error) {
	if v.Val.IsNil() {
		return nil, nil
	}
	// The type switch guarantees each As* call matches the value's type, so
	// their (impossible) errors are discarded.
	switch v.Val.Type() {
	case core.TypeString:
		s, _ := v.Val.AsStr()
		return s, nil
	case core.TypeInt:
		i, _ := v.Val.AsInt()
		return i, nil
	case core.TypeFloat:
		f, _ := v.Val.AsFloat()
		return f, nil
	case core.TypeBool:
		b, _ := v.Val.AsBool()
		if b {
			return int64(1), nil
		}
		return int64(0), nil
	case core.TypeUnsigned:
		u, _ := v.Val.AsUint()
		return int64(u), nil
	case core.TypeTime:
		ts, _ := v.Val.AsTime()
		return ts.UnixMilli(), nil
	case core.TypeJSON:
		j, _ := v.Val.AsJSON()
		return string(j), nil
	case core.TypeBytes:
		b, _ := v.Val.AsBytes()
		return b, nil
	default:
		if v.Val.Type().ID() == core.TIDArray {
			b, err := marshalArray(v.Val)
			if err != nil {
				return nil, err
			}
			return string(b), nil
		}
		return nil, nil
	}
}

// TypeFrom reconstructs a [core.Type] from the stored _type and _elem
// columns of a KV row. For arrays, _elem carries the element type id.
func TypeFrom(tid, elem string) core.Type {
	if core.TID(tid) == core.TIDArray {
		return core.NewArrayType(core.TID(elem))
	}
	return core.NewType(core.TID(tid))
}

// ElemTID returns the element type id to store in a KV row's _elem
// column: the array element for array values, empty otherwise.
func (v Value) ElemTID() string {
	if v.Val != nil && v.Val.Type().ID() == core.TIDArray {
		return string(v.Val.Type().ElemTypeID())
	}
	return ""
}

// Scan implements [sql.Scanner] for reading a typed column value.
func (v *Value) Scan(src any) error {
	if src == nil {
		return nil
	}
	switch v.Type {
	case core.TypeInt:
		v.Val = scanInt(src)
	case core.TypeFloat:
		v.Val = scanFloat(src)
	case core.TypeBool:
		v.Val = scanBool(src)
	case core.TypeUnsigned:
		v.Val = scanUnsigned(src)
	case core.TypeTime:
		v.Val = scanTime(src)
	case core.TypeJSON:
		v.Val = scanJSON(src)
	case core.TypeBytes:
		if b, ok := src.([]byte); ok {
			// The []byte handed to a Scanner is owned by the driver and
			// reused on the next scan, so it must be copied before it is
			// retained in a value.
			v.Val = core.BytesVal(append([]byte(nil), b...))
		}
	default:
		if v.Type.ID() == core.TIDArray {
			val, err := scanArray(v.Type.ElemTypeID(), src)
			if err != nil {
				return err
			}
			v.Val = val
		} else { // TypeString
			v.Val = core.StringVal(fmt.Sprint(src))
		}
	}
	return nil
}

func scanInt(src any) *core.Value {
	switch n := src.(type) {
	case int64:
		return core.IntVal(n)
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return core.IntVal(i)
	}
	return core.IntVal(0)
}

func scanFloat(src any) *core.Value {
	switch n := src.(type) {
	case float64:
		return core.FloatVal(n)
	case int64:
		return core.FloatVal(float64(n))
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return core.FloatVal(f)
	}
	return core.FloatVal(0)
}

func scanBool(src any) *core.Value {
	switch n := src.(type) {
	case int64:
		return core.BoolVal(n != 0)
	case bool:
		return core.BoolVal(n)
	case string:
		return core.BoolVal(n == "1" || n == "true")
	}
	return core.BoolVal(false)
}

func scanUnsigned(src any) *core.Value {
	switch n := src.(type) {
	case int64:
		return core.UintVal(uint64(n))
	case string:
		u, _ := strconv.ParseUint(n, 10, 64)
		return core.UintVal(u)
	}
	return core.UintVal(0)
}

func scanTime(src any) *core.Value {
	switch n := src.(type) {
	case int64:
		return core.TimeVal(time.UnixMilli(n).UTC())
	case string:
		t, _ := time.Parse(time.RFC3339Nano, n)
		return core.TimeVal(t)
	case time.Time:
		return core.TimeVal(n)
	}
	return core.TimeVal(time.Time{})
}

func scanJSON(src any) *core.Value {
	switch n := src.(type) {
	case string:
		return core.JSONVal([]byte(n))
	case []byte:
		return core.JSONVal(n)
	}
	return core.JSONVal(nil)
}

// marshalScalar serializes a scalar Val to its text form. It is the
// per-element encoder for array JSON; scalar columns and KV values are
// stored natively via [Value.Value].
func (v Value) marshalScalar() ([]byte, error) {
	if v.Val.IsNil() {
		return nil, nil
	}
	// The type switch guarantees each As* call matches the value's type, so
	// their (impossible) errors are discarded.
	switch v.Val.Type() {
	case core.TypeString:
		s, _ := v.Val.AsStr()
		return []byte(s), nil
	case core.TypeInt:
		i, _ := v.Val.AsInt()
		return strconv.AppendInt(nil, i, 10), nil
	case core.TypeUnsigned:
		u, _ := v.Val.AsUint()
		return strconv.AppendUint(nil, u, 10), nil
	case core.TypeFloat:
		f, _ := v.Val.AsFloat()
		return strconv.AppendFloat(nil, f, 'g', -1, 64), nil
	case core.TypeBool:
		b, _ := v.Val.AsBool()
		return strconv.AppendBool(nil, b), nil
	case core.TypeTime:
		ts, _ := v.Val.AsTime()
		return strconv.AppendInt(nil, ts.UnixMilli(), 10), nil
	case core.TypeJSON:
		j, _ := v.Val.AsJSON()
		return []byte(j), nil
	case core.TypeBytes:
		b, _ := v.Val.AsBytes()
		return b, nil
	default:
		if v.Val.Type().ID() == core.TIDArray {
			return marshalArray(v.Val)
		}
		return nil, fmt.Errorf("marshal: unsupported type %s", v.Val.Type().ID())
	}
}

// unmarshalScalar deserializes a scalar from its text form into Val.
// It is the per-element decoder for array JSON.
func (v *Value) unmarshalScalar(typ core.Type, data []byte) error {
	s := string(data)
	switch typ {
	case core.TypeString:
		v.Val = core.StringVal(s)
	case core.TypeInt:
		i, _ := strconv.ParseInt(s, 10, 64)
		v.Val = core.IntVal(i)
	case core.TypeUnsigned:
		u, _ := strconv.ParseUint(s, 10, 64)
		v.Val = core.UintVal(u)
	case core.TypeFloat:
		f, _ := strconv.ParseFloat(s, 64)
		v.Val = core.FloatVal(f)
	case core.TypeBool:
		v.Val = core.BoolVal(s == "true")
	case core.TypeTime:
		ms, _ := strconv.ParseInt(s, 10, 64)
		v.Val = core.TimeVal(time.UnixMilli(ms).UTC())
	case core.TypeJSON:
		v.Val = core.JSONVal(data)
	case core.TypeBytes:
		v.Val = core.BytesVal(data)
	default:
		if typ.ID() == core.TIDArray {
			val, err := unmarshalArray(typ.ElemTypeID(), data)
			if err != nil {
				return err
			}
			v.Val = val
			return nil
		}
		return fmt.Errorf("unmarshal: unsupported type %s", typ.ID())
	}
	return nil
}

// marshalArray JSON-encodes array elements by reusing marshalScalar per element.
func marshalArray(v *core.Value) ([]byte, error) {
	elems, _ := v.AsArray()
	parts := make([]json.RawMessage, len(elems))
	for i, e := range elems {
		sv := Value{Val: e}
		b, err := sv.marshalScalar()
		if err != nil {
			return nil, err
		}
		// marshalScalar returns raw text, so wrap strings in JSON quotes.
		if e.Type().ID() == core.TIDString {
			quoted, err := json.Marshal(string(b))
			if err != nil {
				return nil, err
			}
			b = quoted
		}
		parts[i] = b
	}
	return json.Marshal(parts)
}

// scanArray decodes a JSON array from a driver source (string or []byte).
func scanArray(elemTID core.TID, src any) (*core.Value, error) {
	var data []byte
	switch n := src.(type) {
	case string:
		data = []byte(n)
	case []byte:
		data = n
	default:
		return nil, fmt.Errorf("scan array: expected string or []byte, got %T", src)
	}
	return unmarshalArray(elemTID, data)
}

// unmarshalArray decodes a JSON array by reusing unmarshalScalar per element.
func unmarshalArray(elemTID core.TID, data []byte) (*core.Value, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal array: %w", err)
	}

	elemType := core.NewType(elemTID)
	elems := make([]*core.Value, len(raw))
	for i, r := range raw {
		// For strings, JSON has quotes — unmarshal to get the raw string first.
		if elemTID == core.TIDString {
			var s string
			if err := json.Unmarshal(r, &s); err != nil {
				return nil, err
			}
			r = json.RawMessage(s)
		}
		sv := &Value{}
		if err := sv.unmarshalScalar(elemType, r); err != nil {
			return nil, err
		}
		elems[i] = sv.Val
	}
	return core.ArrayVal(elemTID, elems...), nil
}
