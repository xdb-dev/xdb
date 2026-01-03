package tests

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

// FakePostSchema creates a fake schema of a Post.
func FakePostSchema() *schema.Def {
	return &schema.Def{
		Name:        "posts",
		Description: "Blog post schema",
		Version:     "1.0.0",
		Mode:        schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "title", Type: core.TypeString},
			{Name: "content", Type: core.TypeString},
			{Name: "tags", Type: core.NewArrayType(core.TIDString)},
			{Name: "metadata", Type: core.NewMapType(core.TIDString, core.TIDString)},
			{Name: "rating", Type: core.TypeFloat},
			{Name: "published", Type: core.TypeBool},
			{Name: "comments.count", Type: core.TypeInt},
			{Name: "thumbnail", Type: core.TypeBytes},
			{Name: "created_at", Type: core.TypeTime},
		},
	}
}

// FakePost creates a fake record with fake Post data.
func FakePost() *core.Record {
	record := core.NewRecord("com.example", "posts", gofakeit.UUID())

	return record.Set("title", gofakeit.Sentence(10)).
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

// FakeAllTypesSchema creates a fake schema covering all XDB types.
func FakeAllTypesSchema() *schema.Def {
	return &schema.Def{
		Name:        "all_types",
		Description: "Comprehensive test schema covering all types",
		Version:     "1.0.0",
		Mode:        schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "string", Type: core.TypeString},
			{Name: "string_array", Type: core.NewArrayType(core.TIDString)},
			{Name: "int", Type: core.TypeInt},
			{Name: "int64", Type: core.TypeInt},
			{Name: "int64_array", Type: core.NewArrayType(core.TIDInteger)},
			{Name: "unsigned", Type: core.TypeUnsigned},
			{Name: "unsigned_array", Type: core.NewArrayType(core.TIDUnsigned)},
			{Name: "float", Type: core.TypeFloat},
			{Name: "float_array", Type: core.NewArrayType(core.TIDFloat)},
			{Name: "bool", Type: core.TypeBool},
			{Name: "bool_array", Type: core.NewArrayType(core.TIDBoolean)},
			{Name: "bytes", Type: core.TypeBytes},
			{Name: "bytes_array", Type: core.NewArrayType(core.TIDBytes)},
			{Name: "time", Type: core.TypeTime},
			{Name: "time_array", Type: core.NewArrayType(core.TIDTime)},
			{Name: "metadata", Type: core.NewMapType(core.TIDString, core.TIDString)},
		},
	}
}

// FakeAllTypesRecord creates a fake record with comprehensive type coverage.
func FakeAllTypesRecord() *core.Record {
	return core.NewRecord("com.example", "all_types", gofakeit.UUID()).
		Set("string", gofakeit.Sentence(10)).
		Set("string_array", []string{
			gofakeit.Sentence(10),
			gofakeit.Sentence(10),
		}).
		Set("int", gofakeit.Int64()).
		Set("int64", gofakeit.Int64()).
		Set("int64_array", []int64{
			gofakeit.Int64(),
			gofakeit.Int64(),
		}).
		Set("unsigned", gofakeit.Uint64()).
		Set("unsigned_array", []uint64{
			gofakeit.Uint64(),
			gofakeit.Uint64(),
		}).
		Set("float", gofakeit.Float64()).
		Set("float_array", []float64{
			gofakeit.Float64(),
			gofakeit.Float64(),
		}).
		Set("bool", gofakeit.Bool()).
		Set("bool_array", []bool{
			gofakeit.Bool(),
			gofakeit.Bool(),
		}).
		Set("bytes", []byte(gofakeit.Sentence(10))).
		Set("bytes_array", [][]byte{
			[]byte(gofakeit.Sentence(10)),
			[]byte(gofakeit.Sentence(10)),
		}).
		Set("time", gofakeit.Date()).
		Set("time_array", []time.Time{
			gofakeit.Date(),
			gofakeit.Date(),
		}).
		Set("metadata", map[string]string{
			gofakeit.Word(): gofakeit.Sentence(5),
			gofakeit.Word(): gofakeit.Sentence(5),
		})
}

// FakeTuples creates a list of fake tuples covering all core types.
func FakeTuples() []*core.Tuple {
	r := FakeAllTypesRecord()
	return r.Tuples()
}
