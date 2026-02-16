package xdbsqlite

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
)

var (
	ErrValueConversion = errors.New("[xdbsqlite] value conversion error")
	ErrNilFieldType    = errors.New("[xdbsqlite] nil field type")
)

// valueToSQL converts a core.Value to a SQLite-compatible type.
// Returns nil for nil values.
// Arrays and Maps are JSON-encoded to strings.
func valueToSQL(v *core.Value) (any, error) {
	if v == nil || v.IsNil() {
		return nil, nil
	}

	switch v.Type().ID() {
	case core.TIDString:
		return v.Unwrap().(string), nil

	case core.TIDInteger:
		return v.Unwrap().(int64), nil

	case core.TIDUnsigned:
		return int64(v.Unwrap().(uint64)), nil

	case core.TIDBoolean:
		if v.Unwrap().(bool) {
			return int64(1), nil
		}
		return int64(0), nil

	case core.TIDFloat:
		return v.Unwrap().(float64), nil

	case core.TIDBytes:
		return v.Unwrap().([]byte), nil

	case core.TIDTime:
		return v.Unwrap().(time.Time).UnixNano(), nil

	case core.TIDArray:
		return encodeArrayToJSON(v)

	case core.TIDMap:
		return encodeMapToJSON(v)

	default:
		return nil, errors.Wrap(ErrValueConversion, "type", v.Type().ID().String())
	}
}

func encodeArrayToJSON(v *core.Value) (string, error) {
	arr := v.Unwrap().([]*core.Value)
	goArr := make([]any, len(arr))

	for i, elem := range arr {
		goVal, err := valueToGo(elem)
		if err != nil {
			return "", err
		}
		goArr[i] = goVal
	}

	jsonBytes, err := json.Marshal(goArr)
	if err != nil {
		return "", errors.Wrap(ErrValueConversion, "reason", "json marshal failed")
	}

	return string(jsonBytes), nil
}

func encodeMapToJSON(v *core.Value) (string, error) {
	mp := v.Unwrap().(map[*core.Value]*core.Value)
	goMap := make(map[string]any, len(mp))

	for k, val := range mp {
		keyStr := k.String()
		goVal, err := valueToGo(val)
		if err != nil {
			return "", err
		}
		goMap[keyStr] = goVal
	}

	jsonBytes, err := json.Marshal(goMap)
	if err != nil {
		return "", errors.Wrap(ErrValueConversion, "reason", "json marshal failed")
	}

	return string(jsonBytes), nil
}

func valueToGo(v *core.Value) (any, error) {
	if v == nil || v.IsNil() {
		return nil, nil
	}

	switch v.Type().ID() {
	case core.TIDString:
		return v.Unwrap().(string), nil
	case core.TIDInteger:
		return v.Unwrap().(int64), nil
	case core.TIDUnsigned:
		return v.Unwrap().(uint64), nil
	case core.TIDBoolean:
		return v.Unwrap().(bool), nil
	case core.TIDFloat:
		return v.Unwrap().(float64), nil
	case core.TIDBytes:
		return v.Unwrap().([]byte), nil
	case core.TIDTime:
		return v.Unwrap().(time.Time).UnixNano(), nil
	case core.TIDArray:
		arr := v.Unwrap().([]*core.Value)
		goArr := make([]any, len(arr))
		for i, elem := range arr {
			goVal, err := valueToGo(elem)
			if err != nil {
				return nil, err
			}
			goArr[i] = goVal
		}
		return goArr, nil
	case core.TIDMap:
		mp := v.Unwrap().(map[*core.Value]*core.Value)
		goMap := make(map[string]any, len(mp))
		for k, val := range mp {
			goVal, err := valueToGo(val)
			if err != nil {
				return nil, err
			}
			goMap[k.String()] = goVal
		}
		return goMap, nil
	default:
		return nil, errors.Wrap(ErrValueConversion, "type", v.Type().ID().String())
	}
}

// sqlToValue converts a SQLite value back to a core.Value using the field type.
// The fieldType is needed to properly reconstruct typed values.
// JSON strings are decoded for Array/Map types.
func sqlToValue(sqlVal any, fieldType core.Type) (*core.Value, error) {
	if sqlVal == nil {
		return nil, nil
	}

	switch fieldType.ID() {
	case core.TIDString:
		return convertToString(sqlVal)

	case core.TIDInteger:
		return convertToInteger(sqlVal)

	case core.TIDUnsigned:
		return convertToUnsigned(sqlVal)

	case core.TIDBoolean:
		return convertToBoolean(sqlVal)

	case core.TIDFloat:
		return convertToFloat(sqlVal)

	case core.TIDBytes:
		return convertToBytes(sqlVal)

	case core.TIDTime:
		return convertToTime(sqlVal)

	case core.TIDArray:
		return convertToArray(sqlVal, fieldType)

	case core.TIDMap:
		return convertToMap(sqlVal, fieldType)

	default:
		return nil, errors.Wrap(ErrValueConversion, "type", fieldType.ID().String())
	}
}

func convertToString(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case string:
		return core.NewSafeValue(v)
	case []byte:
		return core.NewSafeValue(string(v))
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "string", "got", typeName(sqlVal))
	}
}

func convertToInteger(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case int64:
		return core.NewSafeValue(v)
	case int:
		return core.NewSafeValue(int64(v))
	case int32:
		return core.NewSafeValue(int64(v))
	case float64:
		return core.NewSafeValue(int64(v))
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "integer", "got", typeName(sqlVal))
	}
}

func convertToUnsigned(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case int64:
		return core.NewSafeValue(uint64(v))
	case int:
		return core.NewSafeValue(uint64(v))
	case int32:
		return core.NewSafeValue(uint64(v))
	case float64:
		return core.NewSafeValue(uint64(v))
	case uint64:
		return core.NewSafeValue(v)
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "unsigned", "got", typeName(sqlVal))
	}
}

func convertToBoolean(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case int64:
		return core.NewSafeValue(v != 0)
	case int:
		return core.NewSafeValue(v != 0)
	case int32:
		return core.NewSafeValue(v != 0)
	case bool:
		return core.NewSafeValue(v)
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "boolean", "got", typeName(sqlVal))
	}
}

func convertToFloat(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case float64:
		return core.NewSafeValue(v)
	case float32:
		return core.NewSafeValue(float64(v))
	case int64:
		return core.NewSafeValue(float64(v))
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "float", "got", typeName(sqlVal))
	}
}

func convertToBytes(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case []byte:
		return core.NewSafeValue(v)
	case string:
		return core.NewSafeValue([]byte(v))
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "bytes", "got", typeName(sqlVal))
	}
}

func convertToTime(sqlVal any) (*core.Value, error) {
	switch v := sqlVal.(type) {
	case int64:
		return core.NewSafeValue(time.Unix(0, v).UTC())
	case int:
		return core.NewSafeValue(time.Unix(0, int64(v)).UTC())
	case float64:
		return core.NewSafeValue(time.Unix(0, int64(v)).UTC())
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "time", "got", typeName(sqlVal))
	}
}

func convertToArray(sqlVal any, fieldType core.Type) (*core.Value, error) {
	var jsonStr string

	switch v := sqlVal.(type) {
	case string:
		jsonStr = v
	case []byte:
		jsonStr = string(v)
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "json string", "got", typeName(sqlVal))
	}

	var rawArr []any
	dec := json.NewDecoder(strings.NewReader(jsonStr))
	dec.UseNumber()
	if err := dec.Decode(&rawArr); err != nil {
		return nil, errors.Wrap(ErrValueConversion, "reason", "json unmarshal failed")
	}

	if len(rawArr) == 0 {
		return nil, nil
	}

	elemType := typeForID(fieldType.ValueTypeID())

	arr := make([]*core.Value, len(rawArr))
	for i, elem := range rawArr {
		val, err := goToValue(elem, elemType)
		if err != nil {
			return nil, err
		}
		arr[i] = val
	}

	return core.NewSafeValue(arr)
}

func convertToMap(sqlVal any, fieldType core.Type) (*core.Value, error) {
	var jsonStr string

	switch v := sqlVal.(type) {
	case string:
		jsonStr = v
	case []byte:
		jsonStr = string(v)
	default:
		return nil, errors.Wrap(ErrValueConversion, "expected", "json string", "got", typeName(sqlVal))
	}

	var rawMap map[string]any
	dec := json.NewDecoder(strings.NewReader(jsonStr))
	dec.UseNumber()
	if err := dec.Decode(&rawMap); err != nil {
		return nil, errors.Wrap(ErrValueConversion, "reason", "json unmarshal failed")
	}

	if len(rawMap) == 0 {
		return nil, nil
	}

	keyType := typeForID(fieldType.KeyTypeID())
	if keyType.ID() == core.TIDUnknown {
		keyType = core.TypeString
	}

	valueType := typeForID(fieldType.ValueTypeID())

	mp := make(map[*core.Value]*core.Value, len(rawMap))
	for k, v := range rawMap {
		keyVal, err := goToValue(k, keyType)
		if err != nil {
			return nil, err
		}

		valVal, err := goToValue(v, valueType)
		if err != nil {
			return nil, err
		}

		mp[keyVal] = valVal
	}

	return core.NewSafeValue(mp)
}

func typeForID(tid core.TID) core.Type {
	switch tid {
	case core.TIDString:
		return core.TypeString
	case core.TIDInteger:
		return core.TypeInt
	case core.TIDUnsigned:
		return core.TypeUnsigned
	case core.TIDBoolean:
		return core.TypeBool
	case core.TIDFloat:
		return core.TypeFloat
	case core.TIDBytes:
		return core.TypeBytes
	case core.TIDTime:
		return core.TypeTime
	default:
		return core.Type{}
	}
}

func goToValue(val any, targetType core.Type) (*core.Value, error) {
	if val == nil {
		return nil, nil
	}

	if targetType.ID() == core.TIDUnknown {
		return core.NewSafeValue(val)
	}

	switch targetType.ID() {
	case core.TIDString:
		if s, ok := val.(string); ok {
			return core.NewSafeValue(s)
		}
	case core.TIDInteger:
		return goToInt64Value(val)
	case core.TIDUnsigned:
		return goToUint64Value(val)
	case core.TIDBoolean:
		if b, ok := val.(bool); ok {
			return core.NewSafeValue(b)
		}
	case core.TIDFloat:
		return goToFloat64Value(val)
	case core.TIDBytes:
		if s, ok := val.(string); ok {
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return nil, errors.Wrap(ErrValueConversion, "reason", "base64 decode failed")
			}
			return core.NewSafeValue(b)
		}
	case core.TIDTime:
		return goToTimeValue(val)
	}

	return core.NewSafeValue(val)
}

func goToInt64Value(val any) (*core.Value, error) {
	switch v := val.(type) {
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return nil, errors.Wrap(ErrValueConversion, "reason", "json number to int64 failed")
		}
		return core.NewSafeValue(n)
	case float64:
		return core.NewSafeValue(int64(v))
	case int64:
		return core.NewSafeValue(v)
	}
	return core.NewSafeValue(val)
}

func goToUint64Value(val any) (*core.Value, error) {
	switch v := val.(type) {
	case json.Number:
		n, err := strconv.ParseUint(v.String(), 10, 64)
		if err != nil {
			return nil, errors.Wrap(ErrValueConversion, "reason", "json number to uint64 failed")
		}
		return core.NewSafeValue(n)
	case float64:
		return core.NewSafeValue(uint64(v))
	case uint64:
		return core.NewSafeValue(v)
	}
	return core.NewSafeValue(val)
}

func goToFloat64Value(val any) (*core.Value, error) {
	switch v := val.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return nil, errors.Wrap(ErrValueConversion, "reason", "json number to float64 failed")
		}
		return core.NewSafeValue(f)
	case float64:
		return core.NewSafeValue(v)
	}
	return core.NewSafeValue(val)
}

func goToTimeValue(val any) (*core.Value, error) {
	switch v := val.(type) {
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return nil, errors.Wrap(ErrValueConversion, "reason", "json number to time failed")
		}
		return core.NewSafeValue(time.Unix(0, n).UTC())
	case float64:
		return core.NewSafeValue(time.Unix(0, int64(v)).UTC())
	case int64:
		return core.NewSafeValue(time.Unix(0, v).UTC())
	}
	return core.NewSafeValue(val)
}

func typeName(v any) string {
	if v == nil {
		return "nil"
	}

	switch v.(type) {
	case string:
		return "string"
	case []byte:
		return "[]byte"
	case int:
		return "int"
	case int32:
		return "int32"
	case int64:
		return "int64"
	case uint64:
		return "uint64"
	case float32:
		return "float32"
	case float64:
		return "float64"
	case bool:
		return "bool"
	default:
		return "unknown"
	}
}
