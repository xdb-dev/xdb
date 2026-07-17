package xdbjson_test

import (
	"time"

	"github.com/xdb-dev/xdb/core"
)

// Test-only value extractors that panic on type mismatch, keeping the
// assertions and Example functions terse.

func vStr(v *core.Value) string {
	s, err := v.AsStr()
	if err != nil {
		panic(err)
	}
	return s
}

func vInt(v *core.Value) int64 {
	i, err := v.AsInt()
	if err != nil {
		panic(err)
	}
	return i
}

func vFloat(v *core.Value) float64 {
	f, err := v.AsFloat()
	if err != nil {
		panic(err)
	}
	return f
}

func vBool(v *core.Value) bool {
	b, err := v.AsBool()
	if err != nil {
		panic(err)
	}
	return b
}

func vBytes(v *core.Value) []byte {
	b, err := v.AsBytes()
	if err != nil {
		panic(err)
	}
	return b
}

func vTime(v *core.Value) time.Time {
	t, err := v.AsTime()
	if err != nil {
		panic(err)
	}
	return t
}
