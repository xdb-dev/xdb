package tests

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

// // FakePostSchema creates a fake schema of a Post.
// func FakePostSchema() *core.Schema {
// 	return &core.Schema{
// 		Kind: "Post",
// 		Attributes: []core.Attribute{
// 			{Name: "title", Type: core.NewType(core.TypeIDString)},
// 			{Name: "content", Type: core.NewType(core.TypeIDString)},
// 			{Name: "tags", Type: core.NewArrayType(core.TypeIDString)},
// 			{Name: "rating", Type: core.NewType(core.TypeIDFloat)},
// 			{Name: "published", Type: core.NewType(core.TypeIDBoolean)},
// 			{Name: "comments.count", Type: core.NewType(core.TypeIDInteger)},
// 			{Name: "views.count", Type: core.NewType(core.TypeIDInteger)},
// 			{Name: "likes.count", Type: core.NewType(core.TypeIDInteger)},
// 			{Name: "shares.count", Type: core.NewType(core.TypeIDInteger)},
// 			{Name: "favorites.count", Type: core.NewType(core.TypeIDInteger)},
// 			{Name: "author.id", Type: core.NewType(core.TypeIDString)},
// 			{Name: "author.name", Type: core.NewType(core.TypeIDString)},
// 		},
// 	}
// }

// FakePost creates a fakerecord with fake Post data.
func FakePost() *core.Record {
	kind := "Post"
	id := gofakeit.UUID()

	return core.NewRecord(kind, id).
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
func FakePosts(n int) []*core.Record {
	records := make([]*core.Record, 0, n)

	for i := 0; i < n; i++ {
		records = append(records, FakePost())
	}

	return records
}

// // FakeTupleSchema creates a fake schema for all types of tuples.
// func FakeTupleSchema() *core.Schema {
// 	return &core.Schema{
// 		Kind: "Test",
// 		Attributes: []core.Attribute{
// 			{Name: "id", Type: core.NewType(core.TypeIDString), PrimaryKey: true},
// 			{Name: "string", Type: core.NewType(core.TypeIDString)},
// 			{Name: "int64", Type: core.NewType(core.TypeIDInteger)},
// 			{Name: "float", Type: core.NewType(core.TypeIDFloat)},
// 			{Name: "bool", Type: core.NewType(core.TypeIDBoolean)},
// 			{Name: "bytes", Type: core.NewType(core.TypeIDBytes)},
// 			{Name: "time", Type: core.NewType(core.TypeIDTime)},
// 			{Name: "string_array", Type: core.NewArrayType(core.TypeIDString)},
// 			{Name: "int64_array", Type: core.NewArrayType(core.TypeIDInteger)},
// 			{Name: "float_array", Type: core.NewArrayType(core.TypeIDFloat)},
// 			{Name: "bool_array", Type: core.NewArrayType(core.TypeIDBoolean)},
// 			{Name: "bytes_array", Type: core.NewArrayType(core.TypeIDBytes)},
// 			{Name: "time_array", Type: core.NewArrayType(core.TypeIDTime)},
// 			{Name: "not_found", Type: core.NewType(core.TypeIDString)},
// 		},
// 	}
// }

// FakeTuples creates a list of fake tuples covering all core.
func FakeTuples() []*core.Tuple {
	return x.Join(
		FakeStringTuples(),
		FakeIntTuples(),
		FakeFloatTuples(),
		FakeBoolTuples(),
		FakeBytesTuples(),
	)
}

// FakeStringTuples creates fake tuples for string values.
func FakeStringTuples() []*core.Tuple {
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(id, "string", gofakeit.Sentence(10)),
		core.NewTuple(id, "string_array", []string{
			gofakeit.Sentence(10),
			gofakeit.Sentence(10),
		}),
	}
}

// FakeIntTuples creates fake tuples for int values.
func FakeIntTuples() []*core.Tuple {
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(id, "int64", gofakeit.Int64()),
		core.NewTuple(id, "int64_array", []int64{
			gofakeit.Int64(),
			gofakeit.Int64(),
		}),
	}
}

// FakeFloatTuples creates fake tuples for float values.
func FakeFloatTuples() []*core.Tuple {
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(id, "float", gofakeit.Float64()),
		core.NewTuple(id, "float_array", []float64{
			gofakeit.Float64(),
			gofakeit.Float64(),
		}),
	}
}

// FakeBoolTuples creates fake tuples for bool values.
func FakeBoolTuples() []*core.Tuple {
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(id, "bool", gofakeit.Bool()),
		core.NewTuple(id, "bool_array", []bool{
			gofakeit.Bool(),
			gofakeit.Bool(),
		}),
	}
}

// FakeBytesTuples creates fake tuples for bytes values.
func FakeBytesTuples() []*core.Tuple {
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(id, "bytes", []byte(gofakeit.Sentence(10))),
		core.NewTuple(id, "bytes_array", [][]byte{
			[]byte(gofakeit.Sentence(10)),
			[]byte(gofakeit.Sentence(10)),
		}),
	}
}

// FakeTimeTuples creates fake tuples for time.Time.
func FakeTimeTuples() []*core.Tuple {
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(id, "time", gofakeit.Date()),
		core.NewTuple(id, "time_array", []time.Time{
			gofakeit.Date(),
			gofakeit.Date(),
		}),
	}
}
