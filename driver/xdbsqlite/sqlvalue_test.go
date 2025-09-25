package xdbsqlite

// import (
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"github.com/xdb-dev/xdb/core"
// 	"github.com/xdb-dev/xdb/tests"
// )

// func TestSQLValue(t *testing.T) {
// 	t.Parallel()

// 	// tm := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Truncate(time.Millisecond)
// 	b := []byte("hello")

// 	testcases := []struct {
// 		name  string
// 		value *core.Value
// 		want  any
// 	}{
// 		{
// 			name:  "Bool true",
// 			value: core.NewValue(true),
// 			want:  1,
// 		},
// 		{
// 			name:  "Bool false",
// 			value: core.NewValue(false),
// 			want:  0,
// 		},
// 		{
// 			name:  "Bool Array",
// 			value: core.NewValue([]bool{true, false}),
// 			want:  `[true,false]`,
// 		},
// 		{
// 			name:  "Int64",
// 			value: core.NewValue(int64(42)),
// 			want:  int64(42),
// 		},
// 		{
// 			name:  "Int64 Array",
// 			value: core.NewValue([]int64{1, 2}),
// 			want:  `["1","2"]`,
// 		},
// 		{
// 			name:  "Uint64",
// 			value: core.NewValue(uint64(42)),
// 			want:  uint64(42),
// 		},
// 		{
// 			name:  "Uint64 Array",
// 			value: core.NewValue([]uint64{1, 2}),
// 			want:  `["1","2"]`,
// 		},
// 		{
// 			name:  "Float64",
// 			value: core.NewValue(float64(3.14)),
// 			want:  float64(3.14),
// 		},
// 		{
// 			name:  "Float64 Array",
// 			value: core.NewValue([]float64{1.1, 2.2}),
// 			want:  `[1.1,2.2]`,
// 		},
// 		{
// 			name:  "String",
// 			value: core.NewValue("foo"),
// 			want:  "foo",
// 		},
// 		{
// 			name:  "String Array",
// 			value: core.NewValue([]string{"a", "b"}),
// 			want:  `["a","b"]`,
// 		},
// 		{
// 			name:  "Bytes",
// 			value: core.NewValue(b),
// 			want:  b,
// 		},
// 		{
// 			name:  "Bytes Array",
// 			value: core.NewValue([][]byte{[]byte("a"), []byte("b")}),
// 			want:  `["YQ==","Yg=="]`,
// 		},
// 		// {
// 		// 	name:  "Time",
// 		// 	typ:   core.TimeType{},
// 		// 	value: core.Time(tm),
// 		// 	want:  tm.UnixMilli(),
// 		// },
// 		// {
// 		// 	name: "Time Array",
// 		// 	typ:  core.NewArrayType(core.TimeType{}),
// 		// 	value: core.NewArray(
// 		// 		core.TimeType{},
// 		// 		core.Time(tm),
// 		// 		core.Time(tm.Add(time.Second)),
// 		// 	),
// 		// 	want: `["1718352000000","1718352001000"]`,
// 		// },
// 	}

// 	for _, tc := range testcases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			t.Run("Value", func(t *testing.T) {
// 				sv := &sqlValue{
// 					attr: &core.Attribute{
// 						Type: core.NewType(tc.value.Type().ID()),
// 					},
// 					value: tc.value,
// 				}

// 				got, err := sv.Value()
// 				require.NoError(t, err)
// 				assert.EqualValues(t, tc.want, got)
// 			})

// 			t.Run("Scan", func(t *testing.T) {
// 				sv := &sqlValue{
// 					attr: &core.Attribute{
// 						Type: core.NewType(tc.value.Type().ID()),
// 					},
// 				}

// 				err := sv.Scan(tc.want)
// 				require.NoError(t, err)
// 				tests.AssertEqualValues(t, tc.value, sv.value)
// 			})
// 		})
// 	}
// }
