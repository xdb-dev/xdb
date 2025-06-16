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
			{Name: "title", Type: types.NewType(types.TypeIDString)},
			{Name: "content", Type: types.NewType(types.TypeIDString)},
			{Name: "tags", Type: types.NewArrayType(types.TypeIDString)},
			{Name: "rating", Type: types.NewType(types.TypeIDFloat)},
			{Name: "published", Type: types.NewType(types.TypeIDBoolean)},
			{Name: "comments.count", Type: types.NewType(types.TypeIDInteger)},
			{Name: "views.count", Type: types.NewType(types.TypeIDInteger)},
			{Name: "likes.count", Type: types.NewType(types.TypeIDInteger)},
			{Name: "shares.count", Type: types.NewType(types.TypeIDInteger)},
			{Name: "favorites.count", Type: types.NewType(types.TypeIDInteger)},
			{Name: "author.id", Type: types.NewType(types.TypeIDString)},
			{Name: "author.name", Type: types.NewType(types.TypeIDString)},
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
			{Name: "id", Type: types.NewType(types.TypeIDString), PrimaryKey: true},
			{Name: "string", Type: types.NewType(types.TypeIDString)},
			{Name: "int64", Type: types.NewType(types.TypeIDInteger)},
			{Name: "float", Type: types.NewType(types.TypeIDFloat)},
			{Name: "bool", Type: types.NewType(types.TypeIDBoolean)},
			{Name: "bytes", Type: types.NewType(types.TypeIDBytes)},
			{Name: "time", Type: types.NewType(types.TypeIDTime)},
			{Name: "string_array", Type: types.NewArrayType(types.TypeIDString)},
			{Name: "int64_array", Type: types.NewArrayType(types.TypeIDInteger)},
			{Name: "float_array", Type: types.NewArrayType(types.TypeIDFloat)},
			{Name: "bool_array", Type: types.NewArrayType(types.TypeIDBoolean)},
			{Name: "bytes_array", Type: types.NewArrayType(types.TypeIDBytes)},
			{Name: "time_array", Type: types.NewArrayType(types.TypeIDTime)},
			{Name: "not_found", Type: types.NewType(types.TypeIDString)},
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
