package tests

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

// FakePost creates a fakerecord with fake Post data.
func FakePost() *types.Record {
	kind := "Post"
	id := gofakeit.UUID()

	return types.NewRecord(kind, id).
		Set("title", gofakeit.Sentence(10)).
		Set("content", gofakeit.Paragraph(10, 10, 10, " ")).
		Set("created_at", gofakeit.Date())
}

// FakeTuples creates a list of fake tuples covering all types.
func FakeTuples() []*types.Tuple {
	return x.Join(
		FakeStringTuples(),
		FakeIntTuples(),
		FakeFloatTuples(),
		FakeBoolTuples(),
		FakeBytesTuples(),
	)
}

// FakeStringTuples creates fake tuples for string values.
func FakeStringTuples() []*types.Tuple {
	return []*types.Tuple{
		types.NewTuple("Test", "1", "string", gofakeit.Sentence(10)),
		types.NewTuple("Test", "1", "string_array", []string{
			gofakeit.Sentence(10),
			gofakeit.Sentence(10),
		}),
	}
}

// FakeIntTuples creates fake tuples for int values.
func FakeIntTuples() []*types.Tuple {
	return []*types.Tuple{
		types.NewTuple("Test", "1", "int64", gofakeit.Int64()),
		types.NewTuple("Test", "1", "int64_array", []int64{
			gofakeit.Int64(),
			gofakeit.Int64(),
		}),
	}
}

// FakeFloatTuples creates fake tuples for float values.
func FakeFloatTuples() []*types.Tuple {
	return []*types.Tuple{
		types.NewTuple("Test", "1", "float", gofakeit.Float64()),
		types.NewTuple("Test", "1", "float_array", []float64{
			gofakeit.Float64(),
			gofakeit.Float64(),
		}),
	}
}

// FakeBoolTuples creates fake tuples for bool values.
func FakeBoolTuples() []*types.Tuple {
	return []*types.Tuple{
		types.NewTuple("Test", "1", "bool", gofakeit.Bool()),
		types.NewTuple("Test", "1", "bool_array", []bool{
			gofakeit.Bool(),
			gofakeit.Bool(),
		}),
	}
}

// FakeBytesTuples creates fake tuples for bytes values.
func FakeBytesTuples() []*types.Tuple {
	return []*types.Tuple{
		types.NewTuple("Test", "1", "bytes", []byte(gofakeit.Sentence(10))),
		types.NewTuple("Test", "1", "bytes_array", [][]byte{
			[]byte(gofakeit.Sentence(10)),
			[]byte(gofakeit.Sentence(10)),
		}),
	}
}

// FakeTimeTuples creates fake tuples for time.Time.
func FakeTimeTuples() []*types.Tuple {
	return []*types.Tuple{
		types.NewTuple("Test", "1", "time", gofakeit.Date()),
		types.NewTuple("Test", "1", "time_array", []time.Time{
			gofakeit.Date(),
			gofakeit.Date(),
		}),
	}
}
