package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/types"
)

func TestNewValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := types.NewValue("hello")
		assert.Equal(t, types.StringType, v.TypeID())
		assert.EqualValues(t, "hello", v.Unwrap())
	})

	t.Run("int", func(t *testing.T) {
		v := types.NewValue(1)
		assert.Equal(t, types.IntegerType, v.TypeID())
		assert.EqualValues(t, 1, v.Unwrap())
	})

	t.Run("float", func(t *testing.T) {
		v := types.NewValue(1.0)
		assert.Equal(t, types.FloatType, v.TypeID())
		assert.EqualValues(t, 1.0, v.Unwrap())
	})

	t.Run("bool", func(t *testing.T) {
		v := types.NewValue(true)
		assert.Equal(t, types.BooleanType, v.TypeID())
		assert.EqualValues(t, true, v.Unwrap())
	})

	t.Run("bytes", func(t *testing.T) {
		v := types.NewValue([]byte("hello"))
		assert.Equal(t, types.BytesType, v.TypeID())
		assert.EqualValues(t, []byte("hello"), v.Unwrap())
	})

	t.Run("time", func(t *testing.T) {
		v := types.NewValue(time.Now())
		assert.Equal(t, types.TimeType, v.TypeID())
		assert.EqualValues(t, time.Now(), v.Unwrap())
	})

	t.Run("point", func(t *testing.T) {
		v := types.NewValue(types.Point{Lat: 1.0, Long: 2.0})
		assert.Equal(t, types.PointType, v.TypeID())
		assert.EqualValues(t, types.Point{Lat: 1.0, Long: 2.0}, v.Unwrap())
	})

	t.Run("string slice", func(t *testing.T) {
		v := types.NewValue([]string{"hello", "world"})
		assert.Equal(t, types.StringType, v.TypeID())
		assert.EqualValues(t, []string{"hello", "world"}, v.Unwrap())
	})

	t.Run("unknown", func(t *testing.T) {
		tid, _ := types.TypeIDOf(map[string]string{})
		assert.Equal(t, types.UnknownType, tid)
	})

	t.Run("unknown struct", func(t *testing.T) {
		tid, _ := types.TypeIDOf(struct {
			ID   int
			Name string
		}{})
		assert.Equal(t, types.UnknownType, tid)
	})
}
