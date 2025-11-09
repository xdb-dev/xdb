package tests

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

// FakeRepo creates a fake repository.
func FakeRepo() *core.Repo {
	repo, err := core.NewRepo("com.example.posts")
	if err != nil {
		panic(err)
	}
	return repo.WithSchema(FakePostSchema())
}

// FakePostSchema creates a fake schema of a Post.
func FakePostSchema() *core.Schema {
	return &core.Schema{
		Name:        "com.example.posts",
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

// FakePost creates a fake record with fake Post data.
func FakePost() *core.Record {
	repo := "com.example.posts"
	id := gofakeit.UUID()

	return core.NewRecord(repo, id).
		Set("title", gofakeit.Sentence(10)).
		Set("content", gofakeit.Paragraph(10, 10, 10, " ")).
		Set("tags", []string{
			gofakeit.Word(),
			gofakeit.Word(),
		}).
		Set("metadata", map[string]string{
			"category": gofakeit.Word(),
			"author":   gofakeit.Name(),
		}).
		Set("rating", gofakeit.Float64Range(0, 5)).
		Set("published", gofakeit.Bool()).
		Set("comments.count", gofakeit.IntRange(0, 100)).
		Set("thumbnail", []byte(gofakeit.LoremIpsumSentence(5))).
		Set("created_at", gofakeit.Date())
}

// FakePosts creates a list of fake records.
func FakePosts(n int) []*core.Record {
	records := make([]*core.Record, 0, n)

	for i := 0; i < n; i++ {
		records = append(records, FakePost())
	}

	return records
}

// FakeTestRepo creates a fake repository for comprehensive type testing.
func FakeTestRepo() *core.Repo {
	repo, err := core.NewRepo("test-repo")
	if err != nil {
		panic(err)
	}
	return repo.WithSchema(FakeTestSchema())
}

// FakeTestSchema creates a fake schema covering all XDB types.
func FakeTestSchema() *core.Schema {
	return &core.Schema{
		Name:        "Test",
		Description: "Comprehensive test schema covering all types",
		Version:     "1.0.0",
		Fields: []*core.FieldSchema{
			{Name: "string", Type: core.TypeString},
			{Name: "int", Type: core.TypeInt},
			{Name: "unsigned", Type: core.TypeUnsigned},
			{Name: "float", Type: core.TypeFloat},
			{Name: "bool", Type: core.TypeBool},
			{Name: "bytes", Type: core.TypeBytes},
			{Name: "time", Type: core.TypeTime},
			{Name: "string_array", Type: core.NewArrayType(core.TypeIDString)},
			{Name: "metadata", Type: core.NewMapType(core.TypeIDString, core.TypeIDString)},
		},
		Required: []string{"string"},
	}
}

// FakeTestRecord creates a fake record with comprehensive type coverage.
func FakeTestRecord() *core.Record {
	repo := "test.repo"
	kind := "Test"
	id := gofakeit.UUID()

	return core.NewRecord(repo, kind, id).
		Set("string", gofakeit.Sentence(10)).
		Set("int", gofakeit.Int64()).
		Set("unsigned", gofakeit.Uint64()).
		Set("float", gofakeit.Float64()).
		Set("bool", gofakeit.Bool()).
		Set("bytes", []byte(gofakeit.Sentence(5))).
		Set("time", gofakeit.Date()).
		Set("string_array", []string{gofakeit.Word(), gofakeit.Word()}).
		Set("metadata", map[string]string{
			"key1": gofakeit.Word(),
			"key2": gofakeit.Word(),
		})
}

// FakeTuples creates a list of fake tuples covering all core types.
func FakeTuples() []*core.Tuple {
	return x.Join(
		FakeStringTuples(),
		FakeIntTuples(),
		FakeUnsignedTuples(),
		FakeFloatTuples(),
		FakeBoolTuples(),
		FakeBytesTuples(),
		FakeTimeTuples(),
		FakeMapTuples(),
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

// FakeUnsignedTuples creates fake tuples for unsigned values.
func FakeUnsignedTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "unsigned", gofakeit.Uint64()),
		core.NewTuple(repo, id, "unsigned_array", []uint64{
			gofakeit.Uint64(),
			gofakeit.Uint64(),
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

// FakeMapTuples creates fake tuples for map values.
func FakeMapTuples() []*core.Tuple {
	repo := "test.repo"
	id := core.NewID("Test", "1")
	return []*core.Tuple{
		core.NewTuple(repo, id, "metadata", map[string]string{
			gofakeit.Word(): gofakeit.Sentence(5),
			gofakeit.Word(): gofakeit.Sentence(5),
		}),
	}
}
