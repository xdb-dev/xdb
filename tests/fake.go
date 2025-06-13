package tests

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

// FakePostSchema creates a fake schema of a Post.
func FakePostSchema() *types.Schema {
	return &types.Schema{
		Kind: "Post",
		Attributes: []types.Attribute{
			{Name: "title", Type: types.StringType{}},
			{Name: "content", Type: types.StringType{}},
			{Name: "tags", Type: types.NewArrayType(types.StringType{})},
			{Name: "rating", Type: types.FloatType{}},
			{Name: "published", Type: types.BooleanType{}},
			{Name: "comments.count", Type: types.IntegerType{}},
			{Name: "views.count", Type: types.IntegerType{}},
			{Name: "likes.count", Type: types.IntegerType{}},
			{Name: "shares.count", Type: types.IntegerType{}},
			{Name: "favorites.count", Type: types.IntegerType{}},
			{Name: "author.id", Type: types.StringType{}},
			{Name: "author.name", Type: types.StringType{}},
		},
	}
}

// FakePost creates a fakerecord with fake Post data.
func FakePost() *types.Record {
	kind := "Post"
	id := gofakeit.UUID()

	return types.NewRecord(kind, id).
		Set("title", gofakeit.Sentence(10)).
		Set("content", gofakeit.Paragraph(10, 10, 10, " ")).
		Set("tags", []string{
			gofakeit.Word(),
			gofakeit.Word(),
		}).
		Set("rating", gofakeit.Float64Range(0, 5)).
		Set("published", gofakeit.Bool()).
		Set("comments.count", gofakeit.IntRange(0, 100)).
		Set("views.count", gofakeit.IntRange(0, 1000)).
		Set("likes.count", gofakeit.IntRange(0, 1000)).
		Set("shares.count", gofakeit.IntRange(0, 1000)).
		Set("favorites.count", gofakeit.IntRange(0, 1000)).
		Set("author.id", gofakeit.UUID()).
		Set("author.name", gofakeit.Name())
}

// FakePosts creates a list of fake records.
func FakePosts(n int) []*types.Record {
	records := make([]*types.Record, 0, n)

	for i := 0; i < n; i++ {
		records = append(records, FakePost())
	}

	return records
}

// FakeTupleSchema creates a fake schema for all types of tuples.
func FakeTupleSchema() *types.Schema {
	return &types.Schema{
		Kind: "Test",
		Attributes: []types.Attribute{
			{Name: "id", Type: types.StringType{}, PrimaryKey: true},
			{Name: "string", Type: types.StringType{}},
			{Name: "int64", Type: types.IntegerType{}},
			{Name: "float", Type: types.FloatType{}},
			{Name: "bool", Type: types.BooleanType{}},
			{Name: "bytes", Type: types.BytesType{}},
			{Name: "time", Type: types.TimeType{}},
			{Name: "string_array", Type: types.NewArrayType(types.StringType{})},
			{Name: "int64_array", Type: types.NewArrayType(types.IntegerType{})},
			{Name: "float_array", Type: types.NewArrayType(types.FloatType{})},
			{Name: "bool_array", Type: types.NewArrayType(types.BooleanType{})},
			{Name: "bytes_array", Type: types.NewArrayType(types.BytesType{})},
			{Name: "time_array", Type: types.NewArrayType(types.TimeType{})},
			{Name: "not_found", Type: types.StringType{}},
		},
	}
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
