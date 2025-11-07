package tests

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

// FakeRepo creates a fake repository.
func FakeRepo() *core.Repo {
	repo, err := core.NewRepo("example-repo")
	if err != nil {
		panic(err)
	}
	return repo.WithSchema(FakePostSchema())
}

// FakePostSchema creates a fake schema of a Post.
func FakePostSchema() *core.Schema {
	return &core.Schema{
		Name:        "Post",
		Description: "Blog post schema",
		Version:     "1.0.0",
		Fields: []*core.FieldSchema{
			{Name: "title", Type: core.TypeString},
			{Name: "content", Type: core.TypeString},
			{Name: "tags", Type: core.NewArrayType(core.TypeIDString)},
			{Name: "metadata", Type: core.NewMapType(core.TypeIDString, core.TypeIDString)},
			{Name: "rating", Type: core.TypeFloat},
			{Name: "published", Type: core.TypeBool},
			{Name: "comments.count", Type: core.TypeInt},
			{Name: "thumbnail", Type: core.TypeBytes},
			{Name: "created_at", Type: core.TypeTime},
		},
		Required: []string{"title", "content"},
	}
}

// FakePost creates a fakerecord with fake Post data.
func FakePost() *core.Record {
	repo := "test.repo"
	kind := "Post"
	id := gofakeit.UUID()

	return core.NewRecord(repo, kind, id).
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
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "string", gofakeit.Sentence(10)),
		core.NewTuple(repo, id, "string_array", []string{
			gofakeit.Sentence(10),
			gofakeit.Sentence(10),
		}),
	}
}

// FakeIntTuples creates fake tuples for int values.
func FakeIntTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "int64", gofakeit.Int64()),
		core.NewTuple(repo, id, "int64_array", []int64{
			gofakeit.Int64(),
			gofakeit.Int64(),
		}),
	}
}

// FakeFloatTuples creates fake tuples for float values.
func FakeFloatTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "float", gofakeit.Float64()),
		core.NewTuple(repo, id, "float_array", []float64{
			gofakeit.Float64(),
			gofakeit.Float64(),
		}),
	}
}

// FakeBoolTuples creates fake tuples for bool values.
func FakeBoolTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "bool", gofakeit.Bool()),
		core.NewTuple(repo, id, "bool_array", []bool{
			gofakeit.Bool(),
			gofakeit.Bool(),
		}),
	}
}

// FakeBytesTuples creates fake tuples for bytes values.
func FakeBytesTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "bytes", []byte(gofakeit.Sentence(10))),
		core.NewTuple(repo, id, "bytes_array", [][]byte{
			[]byte(gofakeit.Sentence(10)),
			[]byte(gofakeit.Sentence(10)),
		}),
	}
}

// FakeTimeTuples creates fake tuples for time.Time.
func FakeTimeTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "time", gofakeit.Date()),
		core.NewTuple(repo, id, "time_array", []time.Time{
			gofakeit.Date(),
			gofakeit.Date(),
		}),
	}
}
