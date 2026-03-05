package xdbsqlite

import (
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestSQLValue_Interfaces(t *testing.T) {
	sv := &value{}
	assert.Implements(t, (*driver.Valuer)(nil), sv)
	assert.Implements(t, (*sql.Scanner)(nil), sv)
}

func TestSQLValue_Value_Nil(t *testing.T) {
	sv := &value{}
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSQLValue_Value_String(t *testing.T) {
	v := core.NewValue("hello")
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestSQLValue_Value_Integer(t *testing.T) {
	v := core.NewValue(int64(42))
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(42), result)
}

func TestSQLValue_Value_Unsigned(t *testing.T) {
	v := core.NewValue(uint64(42))
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(42), result)
}

func TestSQLValue_Value_Boolean(t *testing.T) {
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
			sv := newValue(v)
			result, err := sv.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSQLValue_Value_Float(t *testing.T) {
	v := core.NewValue(3.14)
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Equal(t, 3.14, result)
}

func TestSQLValue_Value_Bytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	v := core.NewValue(data)
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestSQLValue_Value_Time(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	v := core.NewValue(now)
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.Equal(t, now.UnixMilli(), result)
}

func TestSQLValue_Value_Array(t *testing.T) {
	arr := []int64{1, 2, 3}
	v := core.NewValue(arr)
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.IsType(t, "", result)
}

func TestSQLValue_Value_Map(t *testing.T) {
	mp := map[string]int64{"a": 1}
	v := core.NewValue(mp)
	sv := newValue(v)
	result, err := sv.Value()
	require.NoError(t, err)
	assert.IsType(t, "", result)
}

func TestSQLValue_Scan_Nil(t *testing.T) {
	sv := newValueFor(core.TypeString)
	err := sv.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, sv.Unwrap())
}

func TestSQLValue_Scan_String(t *testing.T) {
	sv := newValueFor(core.TypeString)
	err := sv.Scan("hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_StringFromBytes(t *testing.T) {
	sv := newValueFor(core.TypeString)
	err := sv.Scan([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_Integer(t *testing.T) {
	sv := newValueFor(core.TypeInt)
	err := sv.Scan(int64(42))
	require.NoError(t, err)
	assert.Equal(t, int64(42), sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_IntegerFromFloat(t *testing.T) {
	sv := newValueFor(core.TypeInt)
	err := sv.Scan(float64(42))
	require.NoError(t, err)
	assert.Equal(t, int64(42), sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_Unsigned(t *testing.T) {
	sv := newValueFor(core.TypeUnsigned)
	err := sv.Scan(int64(42))
	require.NoError(t, err)
	assert.Equal(t, uint64(42), sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_Boolean(t *testing.T) {
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
			sv := newValueFor(core.TypeBool)
			err := sv.Scan(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, sv.Unwrap().Unwrap())
		})
	}
}

func TestSQLValue_Scan_Float(t *testing.T) {
	sv := newValueFor(core.TypeFloat)
	err := sv.Scan(float64(3.14))
	require.NoError(t, err)
	assert.Equal(t, 3.14, sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_Bytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	sv := newValueFor(core.TypeBytes)
	err := sv.Scan(data)
	require.NoError(t, err)
	assert.Equal(t, data, sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_Time(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	sv := newValueFor(core.TypeTime)
	err := sv.Scan(now.UnixMilli())
	require.NoError(t, err)
	assert.Equal(t, now, sv.Unwrap().Unwrap())
}

func TestSQLValue_Scan_Array(t *testing.T) {
	original := core.NewValue([]int64{1, 2, 3})
	encoded, err := newValue(original).Value()
	require.NoError(t, err)

	arrayType := core.NewArrayType(core.TIDInteger)
	sv := newValueFor(arrayType)
	err = sv.Scan(encoded)
	require.NoError(t, err)
	require.NotNil(t, sv.Unwrap())

	arr := sv.Unwrap().Unwrap().([]*core.Value)
	require.Len(t, arr, 3)
	assert.Equal(t, int64(1), arr[0].Unwrap())
	assert.Equal(t, int64(2), arr[1].Unwrap())
	assert.Equal(t, int64(3), arr[2].Unwrap())
}

func TestSQLValue_Scan_Map(t *testing.T) {
	original := core.NewValue(map[string]int64{"a": 1, "b": 2})
	encoded, err := newValue(original).Value()
	require.NoError(t, err)

	mapType := core.NewMapType(core.TIDString, core.TIDInteger)
	sv := newValueFor(mapType)
	err = sv.Scan(encoded)
	require.NoError(t, err)
	require.NotNil(t, sv.Unwrap())

	mp := sv.Unwrap().Unwrap().(map[*core.Value]*core.Value)
	assert.Len(t, mp, 2)
}

func TestSQLValue_RoundTrip_String(t *testing.T) {
	original := core.NewValue("hello world")
	writer := newValue(original)
	sqlVal, err := writer.Value()
	require.NoError(t, err)

	reader := newValueFor(core.TypeString)
	err = reader.Scan(sqlVal)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), reader.Unwrap().Unwrap())
}

func TestSQLValue_RoundTrip_Integer(t *testing.T) {
	original := core.NewValue(int64(-12345))
	writer := newValue(original)
	sqlVal, err := writer.Value()
	require.NoError(t, err)

	reader := newValueFor(core.TypeInt)
	err = reader.Scan(sqlVal)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), reader.Unwrap().Unwrap())
}

func TestSQLValue_RoundTrip_Boolean(t *testing.T) {
	for _, b := range []bool{true, false} {
		original := core.NewValue(b)
		writer := newValue(original)
		sqlVal, err := writer.Value()
		require.NoError(t, err)

		reader := newValueFor(core.TypeBool)
		err = reader.Scan(sqlVal)
		require.NoError(t, err)
		assert.Equal(t, original.Unwrap(), reader.Unwrap().Unwrap())
	}
}

func TestSQLValue_RoundTrip_Float(t *testing.T) {
	original := core.NewValue(3.14159265359)
	writer := newValue(original)
	sqlVal, err := writer.Value()
	require.NoError(t, err)

	reader := newValueFor(core.TypeFloat)
	err = reader.Scan(sqlVal)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), reader.Unwrap().Unwrap())
}

func TestSQLValue_RoundTrip_Bytes(t *testing.T) {
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	original := core.NewValue(data)
	writer := newValue(original)
	sqlVal, err := writer.Value()
	require.NoError(t, err)

	reader := newValueFor(core.TypeBytes)
	err = reader.Scan(sqlVal)
	require.NoError(t, err)
	assert.Equal(t, original.Unwrap(), reader.Unwrap().Unwrap())
}

func TestSQLValue_RoundTrip_Time(t *testing.T) {
	now := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	original := core.NewValue(now)
	writer := newValue(original)
	sqlVal, err := writer.Value()
	require.NoError(t, err)

	reader := newValueFor(core.TypeTime)
	err = reader.Scan(sqlVal)
	require.NoError(t, err)
	assert.Equal(t, now.UnixMilli(), reader.Unwrap().Unwrap().(time.Time).UnixMilli())
}

func TestSQLValue_RoundTrip_Array(t *testing.T) {
	arr := []string{"a", "b", "c"}
	original := core.NewValue(arr)
	writer := newValue(original)
	sqlVal, err := writer.Value()
	require.NoError(t, err)

	arrayType := core.NewArrayType(core.TIDString)
	reader := newValueFor(arrayType)
	err = reader.Scan(sqlVal)
	require.NoError(t, err)

	restoredArr := reader.Unwrap().Unwrap().([]*core.Value)
	require.Len(t, restoredArr, 3)
	assert.Equal(t, "a", restoredArr[0].Unwrap())
	assert.Equal(t, "b", restoredArr[1].Unwrap())
	assert.Equal(t, "c", restoredArr[2].Unwrap())
}

func TestSQLValue_Scan_Error_InvalidType(t *testing.T) {
	sv := newValueFor(core.TypeInt)
	err := sv.Scan("not a number")
	assert.Error(t, err)
}

func TestSQLValue_Scan_Error_InvalidArrayJSON(t *testing.T) {
	arrayType := core.NewArrayType(core.TIDInteger)
	sv := newValueFor(arrayType)
	err := sv.Scan("not valid json")
	assert.Error(t, err)
}

func TestSQLValue_Scan_Error_InvalidMapJSON(t *testing.T) {
	mapType := core.NewMapType(core.TIDString, core.TIDInteger)
	sv := newValueFor(mapType)
	err := sv.Scan("not valid json")
	assert.Error(t, err)
}
