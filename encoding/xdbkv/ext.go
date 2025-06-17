package xdbkv

import (
	"fmt"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/xdb-dev/xdb/types"
)

// value is the wire format for a types.Value
type value struct {
	TypeID types.TypeID `msgpack:"t"`
	Data   any          `msgpack:"d"`
}

// MarshalValue serializes a *types.Value to msgpack.
func MarshalValue(v *types.Value) ([]byte, error) {
	if v == nil || v.IsNil() {
		return msgpack.Marshal(value{TypeID: types.TypeIDUnknown, Data: nil})
	}

	switch v.Type().ID() {
	case types.TypeIDArray:
		arr := v.Unwrap().([]*types.Value)
		data := make([][]byte, len(arr))
		for i, elem := range arr {
			b, err := MarshalValue(elem)
			if err != nil {
				return nil, err
			}
			data[i] = b
		}
		return msgpack.Marshal(value{TypeID: types.TypeIDArray, Data: data})
	case types.TypeIDMap:
		mp := v.Unwrap().(map[*types.Value]*types.Value)
		data := make(map[string][]byte, len(mp))
		for k, val := range mp {
			kb, err := MarshalValue(k)
			if err != nil {
				return nil, err
			}
			vb, err := MarshalValue(val)
			if err != nil {
				return nil, err
			}
			data[string(kb)] = vb
		}
		return msgpack.Marshal(value{TypeID: types.TypeIDMap, Data: data})
	case types.TypeIDTime:
		return msgpack.Marshal(value{TypeID: types.TypeIDTime, Data: v.Unwrap().(time.Time).Format(time.RFC3339)})
	default:
		return msgpack.Marshal(value{TypeID: v.Type().ID(), Data: v.Unwrap()})
	}
}

// UnmarshalValue deserializes msgpack data into a *types.Value.
func UnmarshalValue(b []byte) (*types.Value, error) {
	var env value
	if err := msgpack.Unmarshal(b, &env); err != nil {
		return nil, err
	}

	if env.TypeID == types.TypeIDUnknown || env.Data == nil {
		return nil, nil
	}

	switch env.TypeID {
	case types.TypeIDArray:
		rawArr, ok := env.Data.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid array encoding")
		}

		arr := make([]*types.Value, len(rawArr))
		for i, elem := range rawArr {
			elemBytes, ok := elem.([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid array element encoding")
			}
			v, err := UnmarshalValue(elemBytes)
			if err != nil {
				return nil, err
			}
			arr[i] = v
		}
		return types.NewSafeValue(arr)
	case types.TypeIDMap:
		rawMap, ok := env.Data.(map[string][]byte)
		if !ok {
			return nil, fmt.Errorf("invalid map encoding")
		}
		mp := make(map[*types.Value]*types.Value, len(rawMap))
		for kStr, vBytes := range rawMap {
			kVal, err := UnmarshalValue([]byte(kStr))
			if err != nil {
				return nil, err
			}
			vVal, err := UnmarshalValue(vBytes)
			if err != nil {
				return nil, err
			}
			mp[kVal] = vVal
		}
		return types.NewSafeValue(mp)
	case types.TypeIDTime:
		str, ok := env.Data.(string)
		if !ok {
			return nil, fmt.Errorf("invalid time encoding")
		}
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return nil, err
		}
		return types.NewSafeValue(t)
	default:
		return types.NewSafeValue(env.Data)
	}
}
