package xdbsqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/types"
)

func TestSQLValue(t *testing.T) {
	t.Parallel()

	// tm := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(time.Millisecond)
	b := []byte("hello")

	testcases := []struct {
		name  string
		typ   types.Type
		value types.Value
		want  any
	}{
		{
			name:  "Bool true",
			typ:   types.BooleanType{},
			value: types.Bool(true),
			want:  1,
		},
		{
			name:  "Bool false",
			typ:   types.BooleanType{},
			value: types.Bool(false),
			want:  0,
		},
		{
			name: "Bool Array",
			typ:  types.NewArrayType(types.BooleanType{}),
			value: types.NewArray(
				types.BooleanType{},
				types.Bool(true),
				types.Bool(false),
			),
			want: `[true,false]`,
		},
		{
			name:  "Int64",
			typ:   types.IntegerType{},
			value: types.Int64(42),
			want:  int64(42),
		},
		{
			name: "Int64 Array",
			typ:  types.NewArrayType(types.IntegerType{}),
			value: types.NewArray(
				types.IntegerType{},
				types.Int64(1),
				types.Int64(2),
			),
			want: `["1","2"]`,
		},
		{
			name:  "Uint64",
			typ:   types.UnsignedType{},
			value: types.Uint64(42),
			want:  uint64(42),
		},
		{
			name: "Uint64 Array",
			typ:  types.NewArrayType(types.UnsignedType{}),
			value: types.NewArray(
				types.UnsignedType{},
				types.Uint64(1),
				types.Uint64(2),
			),
			want: `["1","2"]`,
		},
		{
			name:  "Float64",
			typ:   types.FloatType{},
			value: types.Float64(3.14),
			want:  float64(3.14),
		},
		{
			name: "Float64 Array",
			typ:  types.NewArrayType(types.FloatType{}),
			value: types.NewArray(
				types.FloatType{},
				types.Float64(1.1),
				types.Float64(2.2),
			),
			want: `[1.1,2.2]`,
		},
		{
			name:  "String",
			typ:   types.StringType{},
			value: types.String("foo"),
			want:  "foo",
		},
		{
			name: "String Array",
			typ:  types.NewArrayType(types.StringType{}),
			value: types.NewArray(
				types.StringType{},
				types.String("a"),
				types.String("b"),
			),
			want: `["a","b"]`,
		},
		{
			name:  "Bytes",
			typ:   types.BytesType{},
			value: types.Bytes(b),
			want:  b,
		},
		{
			name: "Bytes Array",
			typ:  types.NewArrayType(types.BytesType{}),
			value: types.NewArray(
				types.BytesType{},
				types.Bytes([]byte("a")),
				types.Bytes([]byte("b")),
			),
			want: `["YQ==","Yg=="]`,
		},
		// {
		// 	name:  "Time",
		// 	typ:   types.TimeType{},
		// 	value: types.Time(tm),
		// 	want:  tm.UnixMilli(),
		// },
		// {
		// 	name: "Time Array",
		// 	typ:  types.NewArrayType(types.TimeType{}),
		// 	value: types.NewArray(
		// 		types.TimeType{},
		// 		types.Time(tm),
		// 		types.Time(tm.Add(time.Second)),
		// 	),
		// 	want: `["1718352000000","1718352001000"]`,
		// },
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("Value", func(t *testing.T) {
				sv := &sqlValue{
					attr:  &types.Attribute{Type: tc.typ},
					value: tc.value,
				}

				got, err := sv.Value()
				require.NoError(t, err)
				assert.EqualValues(t, tc.want, got)
			})

			t.Run("Scan", func(t *testing.T) {
				sv := &sqlValue{
					attr: &types.Attribute{Type: tc.typ},
				}

				err := sv.Scan(tc.want)
				require.NoError(t, err)
				tests.AssertEqualValues(t, tc.value, sv.value)
			})
		})
	}
}
