package xdbredis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Type prefixes for encoding values as self-describing strings.
const (
	prefixBool     = "b:"
	prefixInt      = "i:"
	prefixUnsigned = "u:"
	prefixFloat    = "f:"
	prefixString   = "s:"
	prefixBytes    = "x:"
	prefixTime     = "t:"
	prefixJSON     = "j:"
	prefixArray    = "a:"
)

// encodeValue encodes a [core.Value] into a type-prefixed string.
func encodeValue(v *core.Value) (string, error) {
	switch v.Type().ID() {
	case core.TIDBoolean:
		return encodeBool(v)
	case core.TIDInteger:
		return encodeInt(v)
	case core.TIDUnsigned:
		return encodeUint(v)
	case core.TIDFloat:
		return encodeFloat(v)
	case core.TIDString:
		return encodeStr(v)
	case core.TIDBytes:
		return encodeBytes(v)
	case core.TIDTime:
		return encodeTime(v)
	case core.TIDJSON:
		return encodeJSON(v)
	case core.TIDArray:
		return encodeArray(v)
	default:
		return "", fmt.Errorf("xdbredis: unsupported type %s", v.Type().ID())
	}
}

func encodeBool(v *core.Value) (string, error) {
	b, err := v.AsBool()
	if err != nil {
		return "", err
	}
	return prefixBool + strconv.FormatBool(b), nil
}

func encodeInt(v *core.Value) (string, error) {
	i, err := v.AsInt()
	if err != nil {
		return "", err
	}
	return prefixInt + strconv.FormatInt(i, 10), nil
}

func encodeUint(v *core.Value) (string, error) {
	u, err := v.AsUint()
	if err != nil {
		return "", err
	}
	return prefixUnsigned + strconv.FormatUint(u, 10), nil
}

func encodeFloat(v *core.Value) (string, error) {
	f, err := v.AsFloat()
	if err != nil {
		return "", err
	}
	return prefixFloat + strconv.FormatFloat(f, 'f', -1, 64), nil
}

func encodeStr(v *core.Value) (string, error) {
	s, err := v.AsStr()
	if err != nil {
		return "", err
	}
	return prefixString + s, nil
}

func encodeBytes(v *core.Value) (string, error) {
	b, err := v.AsBytes()
	if err != nil {
		return "", err
	}
	return prefixBytes + base64.StdEncoding.EncodeToString(b), nil
}

func encodeTime(v *core.Value) (string, error) {
	t, err := v.AsTime()
	if err != nil {
		return "", err
	}
	return prefixTime + t.Format(time.RFC3339Nano), nil
}

func encodeJSON(v *core.Value) (string, error) {
	j, err := v.AsJSON()
	if err != nil {
		return "", err
	}
	return prefixJSON + string(j), nil
}

// encodeArray encodes an array value as a JSON object with type metadata.
func encodeArray(v *core.Value) (string, error) {
	elems, err := v.AsArray()
	if err != nil {
		return "", err
	}

	encoded := make([]string, len(elems))
	for i, elem := range elems {
		enc, encErr := encodeValue(elem)
		if encErr != nil {
			return "", fmt.Errorf("xdbredis: encode array element %d: %w", i, encErr)
		}
		encoded[i] = enc
	}

	wrapper := arrayEnvelope{
		ElemType: string(v.Type().ElemTypeID()),
		Items:    encoded,
	}

	data, err := json.Marshal(wrapper)
	if err != nil {
		return "", fmt.Errorf("xdbredis: marshal array: %w", err)
	}

	return prefixArray + string(data), nil
}

// arrayEnvelope wraps array data with element type metadata for decoding.
type arrayEnvelope struct {
	ElemType string   `json:"et"`
	Items    []string `json:"items"`
}

// decodeValue decodes a type-prefixed string back into a [core.Value].
func decodeValue(s string) (*core.Value, error) {
	if len(s) < 2 {
		return nil, fmt.Errorf("xdbredis: value too short: %q", s)
	}

	prefix := s[:2]
	payload := s[2:]

	switch prefix {
	case prefixBool:
		return decodeBool(payload)
	case prefixInt:
		return decodeInt(payload)
	case prefixUnsigned:
		return decodeUnsigned(payload)
	case prefixFloat:
		return decodeFloat(payload)
	case prefixString:
		return core.StringVal(payload), nil
	case prefixBytes:
		return decodeBytes(payload)
	case prefixTime:
		return decodeTime(payload)
	case prefixJSON:
		return core.JSONVal(json.RawMessage(payload)), nil
	case prefixArray:
		return decodeArray(payload)
	default:
		return nil, fmt.Errorf("xdbredis: unknown type prefix %q", prefix)
	}
}

func decodeBool(payload string) (*core.Value, error) {
	b, err := strconv.ParseBool(payload)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode bool: %w", err)
	}
	return core.BoolVal(b), nil
}

func decodeInt(payload string) (*core.Value, error) {
	i, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode int: %w", err)
	}
	return core.IntVal(i), nil
}

func decodeUnsigned(payload string) (*core.Value, error) {
	u, err := strconv.ParseUint(payload, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode uint: %w", err)
	}
	return core.UintVal(u), nil
}

func decodeFloat(payload string) (*core.Value, error) {
	f, err := strconv.ParseFloat(payload, 64)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode float: %w", err)
	}
	return core.FloatVal(f), nil
}

func decodeBytes(payload string) (*core.Value, error) {
	b, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode bytes: %w", err)
	}
	return core.BytesVal(b), nil
}

func decodeTime(payload string) (*core.Value, error) {
	t, err := time.Parse(time.RFC3339Nano, payload)
	if err != nil {
		return nil, fmt.Errorf("xdbredis: decode time: %w", err)
	}
	return core.TimeVal(t), nil
}

// decodeArray decodes an array payload into a [core.Value].
func decodeArray(payload string) (*core.Value, error) {
	var env arrayEnvelope
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		return nil, fmt.Errorf("xdbredis: unmarshal array: %w", err)
	}

	elemTID := core.TID(env.ElemType)

	elems := make([]*core.Value, len(env.Items))
	for i, item := range env.Items {
		v, err := decodeValue(item)
		if err != nil {
			return nil, fmt.Errorf("xdbredis: decode array element %d: %w", i, err)
		}
		elems[i] = v
	}

	return core.ArrayVal(elemTID, elems...), nil
}
