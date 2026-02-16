package xdbsqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestValueToSQL_NilValue(t *testing.T) {
	result, err := valueToSQL(nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestValueToSQL_String(t *testing.T) {
	v := core.NewValue("hello")
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestValueToSQL_Integer(t *testing.T) {
	v := core.NewValue(int64(42))
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result)
}

func TestValueToSQL_Unsigned(t *testing.T) {
	v := core.NewValue(uint64(42))
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result)
}

func TestValueToSQL_Boolean(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected int64
	}{
		{"true", true, 1},
		{"false", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := core.NewValue(tt.input)
			result, err := valueToSQL(v)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValueToSQL_Float(t *testing.T) {
	v := core.NewValue(3.14)
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, 3.14, result)
}

func TestValueToSQL_Bytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	v := core.NewValue(data)
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestValueToSQL_Time(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	v := core.NewValue(now)
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, now.UnixNano(), result)
}

func TestValueToSQL_Array(t *testing.T) {
	arr := []int64{1, 2, 3}
	v := core.NewValue(arr)
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, "[1,2,3]", result)
}

func TestValueToSQL_Map(t *testing.T) {
	mp := map[string]int64{"a": 1}
	v := core.NewValue(mp)
	result, err := valueToSQL(v)
	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, result)
}

func TestSQLToValue_NilValue(t *testing.T) {
	result, err := sqlToValue(nil, core.TypeString)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSQLToValue_String(t *testing.T) {
	result, err := sqlToValue("hello", core.TypeString)
	require.NoError(t, err)
	assert.Equal(t, "hello", result.Unwrap())
}

func TestSQLToValue_StringFromBytes(t *testing.T) {
	result, err := sqlToValue([]byte("hello"), core.TypeString)
	require.NoError(t, err)
	assert.Equal(t, "hello", result.Unwrap())
}

func TestSQLToValue_Integer(t *testing.T) {
	result, err := sqlToValue(int64(42), core.TypeInt)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result.Unwrap())
}

func TestSQLToValue_IntegerFromFloat(t *testing.T) {
	result, err := sqlToValue(float64(42), core.TypeInt)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result.Unwrap())
}

func TestSQLToValue_Unsigned(t *testing.T) {
	result, err := sqlToValue(int64(42), core.TypeUnsigned)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), result.Unwrap())
}

func TestSQLToValue_Boolean(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected bool
	}{
		{"true", 1, true},
		{"false", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sqlToValue(tt.input, core.TypeBool)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Unwrap())
		})
	}
}

func TestSQLToValue_Float(t *testing.T) {
	result, err := sqlToValue(float64(3.14), core.TypeFloat)
	require.NoError(t, err)
	assert.Equal(t, 3.14, result.Unwrap())
}

func TestSQLToValue_Bytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	result, err := sqlToValue(data, core.TypeBytes)
	require.NoError(t, err)
	assert.Equal(t, data, result.Unwrap())
}

func TestSQLToValue_Time(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result, err := sqlToValue(now.UnixNano(), core.TypeTime)
	require.NoError(t, err)
	assert.Equal(t, now, result.Unwrap())
}

func TestSQLToValue_Array(t *testing.T) {
	arrayType := core.NewArrayType(core.TIDInteger)
	result, err := sqlToValue("[1,2,3]", arrayType)
	require.NoError(t, err)
	require.NotNil(t, result)

	arr := result.Unwrap().([]*core.Value)
	require.Len(t, arr, 3)
	assert.Equal(t, int64(1), arr[0].Unwrap())
	assert.Equal(t, int64(2), arr[1].Unwrap())
	assert.Equal(t, int64(3), arr[2].Unwrap())
}

func TestSQLToValue_Map(t *testing.T) {
	mapType := core.NewMapType(core.TIDString, core.TIDInteger)
	result, err := sqlToValue(`{"a":1,"b":2}`, mapType)
	require.NoError(t, err)
	require.NotNil(t, result)

	mp := result.Unwrap().(map[*core.Value]*core.Value)
	assert.Len(t, mp, 2)
}

func TestValueRoundTrip_String(t *testing.T) {
	original := core.NewValue("hello world")
	sqlVal, err := valueToSQL(original)
	require.NoError(t, err)

	restored, err := sqlToValue(sqlVal, core.TypeString)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), restored.Unwrap())
}

func TestValueRoundTrip_Integer(t *testing.T) {
	original := core.NewValue(int64(-12345))
	sqlVal, err := valueToSQL(original)
	require.NoError(t, err)

	restored, err := sqlToValue(sqlVal, core.TypeInt)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), restored.Unwrap())
}

func TestValueRoundTrip_Boolean(t *testing.T) {
	for _, b := range []bool{true, false} {
		original := core.NewValue(b)
		sqlVal, err := valueToSQL(original)
		require.NoError(t, err)

		restored, err := sqlToValue(sqlVal, core.TypeBool)
		require.NoError(t, err)
		assert.Equal(t, original.Unwrap(), restored.Unwrap())
	}
}

func TestValueRoundTrip_Float(t *testing.T) {
	original := core.NewValue(3.14159265359)
	sqlVal, err := valueToSQL(original)
	require.NoError(t, err)

	restored, err := sqlToValue(sqlVal, core.TypeFloat)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), restored.Unwrap())
}

func TestValueRoundTrip_Bytes(t *testing.T) {
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	original := core.NewValue(data)
	sqlVal, err := valueToSQL(original)
	require.NoError(t, err)

	restored, err := sqlToValue(sqlVal, core.TypeBytes)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), restored.Unwrap())
}

func TestValueRoundTrip_Time(t *testing.T) {
	now := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	original := core.NewValue(now)
	sqlVal, err := valueToSQL(original)
	require.NoError(t, err)

	restored, err := sqlToValue(sqlVal, core.TypeTime)
	require.NoError(t, err)
	assert.Equal(t, now.UnixMilli(), restored.Unwrap().(time.Time).UnixMilli())
}

func TestValueRoundTrip_Array(t *testing.T) {
	arr := []string{"a", "b", "c"}
	original := core.NewValue(arr)
	sqlVal, err := valueToSQL(original)
	require.NoError(t, err)

	arrayType := core.NewArrayType(core.TIDString)
	restored, err := sqlToValue(sqlVal, arrayType)
	require.NoError(t, err)

	restoredArr := restored.Unwrap().([]*core.Value)
	require.Len(t, restoredArr, 3)
	assert.Equal(t, "a", restoredArr[0].Unwrap())
	assert.Equal(t, "b", restoredArr[1].Unwrap())
	assert.Equal(t, "c", restoredArr[2].Unwrap())
}

func TestConversionError_InvalidType(t *testing.T) {
	_, err := sqlToValue("not a number", core.TypeInt)
	assert.Error(t, err)
}

func TestConversionError_InvalidArrayJSON(t *testing.T) {
	arrayType := core.NewArrayType(core.TIDInteger)
	_, err := sqlToValue("not valid json", arrayType)
	assert.Error(t, err)
}

func TestConversionError_InvalidMapJSON(t *testing.T) {
	mapType := core.NewMapType(core.TIDString, core.TIDInteger)
	_, err := sqlToValue("not valid json", mapType)
	assert.Error(t, err)
}
