package x

import "github.com/spf13/cast"

// CastArray casts an array of any to an array of T.
func CastArray[T any](v []any, f func(any) T) []T {
	arr := make([]T, len(v))
	for i, v := range v {
		arr[i] = f(v)
	}
	return arr
}

// ToBytes converts a value to a byte slice.
func ToBytes(v any) []byte {
	switch v := v.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	default:
		return []byte(cast.ToString(v))
	}
}
