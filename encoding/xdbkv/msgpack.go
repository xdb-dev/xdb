package xdbkv

import (
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/xdb-dev/xdb/types"
)

// value is the wire format for a types.Value
type value struct {
	TypeID types.TypeID `msgpack:"t"`
	Data   any          `msgpack:"d"`
}

func marshalValue(v *types.Value) ([]byte, error) {
	if v == nil || v.IsNil() {
		return msgpack.Marshal(value{TypeID: types.TypeIDUnknown, Data: nil})
	}

	switch v.Type().ID() {
	case types.TypeIDArray:
		arr := v.Unwrap().([]*types.Value)
		data := make([][]byte, len(arr))

		for i, elem := range arr {
			b, err := marshalValue(elem)
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
			kb, err := marshalValue(k)
			if err != nil {
				return nil, err
			}

			vb, err := marshalValue(val)
			if err != nil {
				return nil, err
			}

			data[string(kb)] = vb
		}

		return msgpack.Marshal(value{TypeID: types.TypeIDMap, Data: data})
	case types.TypeIDTime:
		unixtime := v.Unwrap().(time.Time).UnixMilli()

		return msgpack.Marshal(value{TypeID: types.TypeIDTime, Data: unixtime})
	default:
		return msgpack.Marshal(value{TypeID: v.Type().ID(), Data: v.Unwrap()})
	}
}

func unmarshalValue(b []byte) (*types.Value, error) {
	var decoded value
	if err := msgpack.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}

	if decoded.TypeID == types.TypeIDUnknown || decoded.Data == nil {
		return nil, nil
	}

	switch decoded.TypeID {
	case types.TypeIDArray:
		rawArr, ok := decoded.Data.([]any)
		if !ok {
			return nil, ErrDecodingValue
		}

		arr := make([]*types.Value, len(rawArr))
		for i, elem := range rawArr {
			elemBytes, ok := elem.([]byte)
			if !ok {
				return nil, ErrDecodingValue
			}
			v, err := unmarshalValue(elemBytes)
			if err != nil {
				return nil, err
			}
			arr[i] = v
		}
		return types.NewSafeValue(arr)
	case types.TypeIDMap:
		rawMap, ok := decoded.Data.(map[string][]byte)
		if !ok {
			return nil, ErrDecodingValue
		}
		mp := make(map[*types.Value]*types.Value, len(rawMap))
		for kStr, vBytes := range rawMap {
			kVal, err := unmarshalValue([]byte(kStr))
			if err != nil {
				return nil, err
			}
			vVal, err := unmarshalValue(vBytes)
			if err != nil {
				return nil, err
			}
			mp[kVal] = vVal
		}
		return types.NewSafeValue(mp)
	case types.TypeIDTime:
		unixtime, ok := decoded.Data.(int64)
		if !ok {
			return nil, ErrDecodingValue
		}

		t := time.UnixMilli(unixtime).UTC()

		return types.NewSafeValue(t)
	default:
		return types.NewSafeValue(decoded.Data)
	}
}
