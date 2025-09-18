package xdbstruct_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/encoding/xdbstruct"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/core"
)

type IntTypesStruct struct {
	ID string `xdb:"id,primary_key"`

	// Integer Types
	Int   int   `xdb:"int"`
	Int8  int8  `xdb:"int8"`
	Int16 int16 `xdb:"int16"`
	Int32 int32 `xdb:"int32"`
	Int64 int64 `xdb:"int64"`

	// IntegerArray
	IntArray [3]int `xdb:"int_array"`

	// Unsigned Integer Types
	Uint   uint   `xdb:"uint"`
	Uint8  uint8  `xdb:"uint8"`
	Uint16 uint16 `xdb:"uint16"`
	Uint32 uint32 `xdb:"uint32"`
	Uint64 uint64 `xdb:"uint64"`

	// IntegerArray
	UintArray [3]uint `xdb:"uint_array"`
}

func TestIntTypesStruct(t *testing.T) {
	t.Parallel()

	input := IntTypesStruct{
		ID:  "123-456-789",
		Int: 1, Int8: 1, Int16: 1, Int32: 1, Int64: 1,
		IntArray: [3]int{1, 2, 3},
		Uint:     1, Uint8: 1, Uint16: 1, Uint32: 1, Uint64: 1,
		UintArray: [3]uint{1, 2, 3},
	}
	record := core.NewRecord("IntTypesStruct", input.ID).
		Set("int", input.Int).
		Set("int8", input.Int8).
		Set("int16", input.Int16).
		Set("int32", input.Int32).
		Set("int64", input.Int64).
		Set("int_array", input.IntArray).
		Set("uint", input.Uint).
		Set("uint8", input.Uint8).
		Set("uint16", input.Uint16).
		Set("uint32", input.Uint32).
		Set("uint64", input.Uint64).
		Set("uint_array", input.UintArray)

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type FloatTypesStruct struct {
	ID         string     `xdb:"id,primary_key"`
	Float32    float32    `xdb:"float32"`
	Float64    float64    `xdb:"float64"`
	FloatArray [3]float64 `xdb:"float_array"`
}

func TestFloatTypesStruct(t *testing.T) {
	t.Parallel()

	input := FloatTypesStruct{
		ID:         "123-456-789",
		Float32:    123.456,
		Float64:    123.456,
		FloatArray: [3]float64{123.456, 123.456, 123.456},
	}
	record := core.NewRecord("FloatTypesStruct", input.ID).
		Set("float32", input.Float32).
		Set("float64", input.Float64).
		Set("float_array", input.FloatArray)

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type StringTypesStruct struct {
	ID          string    `xdb:"id,primary_key"`
	String      string    `xdb:"string"`
	StringArray [3]string `xdb:"string_array"`
}

func TestStringTypesStruct(t *testing.T) {
	t.Parallel()

	input := StringTypesStruct{
		ID:          "123-456-789",
		String:      "Hello, World!",
		StringArray: [3]string{"Hello", "World", "!"},
	}
	record := core.NewRecord("StringTypesStruct", input.ID).
		Set("string", input.String).
		Set("string_array", input.StringArray)

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type BoolTypesStruct struct {
	ID        string  `xdb:"id,primary_key"`
	Bool      bool    `xdb:"bool"`
	BoolArray [3]bool `xdb:"bool_array"`
}

func TestBoolTypesStruct(t *testing.T) {
	t.Parallel()

	input := BoolTypesStruct{
		ID:        "123-456-789",
		Bool:      true,
		BoolArray: [3]bool{true, false, true},
	}
	record := core.NewRecord("BoolTypesStruct", input.ID).
		Set("bool", input.Bool).
		Set("bool_array", input.BoolArray)

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type ByteTypesStruct struct {
	ID         string    `xdb:"id,primary_key"`
	Bytes      []byte    `xdb:"bytes"`
	BytesArray [3][]byte `xdb:"bytes_array"`

	// JSON
	JSON      json.RawMessage    `xdb:"json"`
	JSONArray [3]json.RawMessage `xdb:"json_array"`

	// Binary
	Binary      *Binary    `xdb:"binary"`
	BinaryArray [3]*Binary `xdb:"binary_array"`
}

type Binary struct {
	Data []byte
}

func (b *Binary) MarshalBinary() ([]byte, error) {
	return b.Data, nil
}

func (b *Binary) UnmarshalBinary(data []byte) error {
	b.Data = data
	return nil
}

func TestByteTypesStruct(t *testing.T) {
	t.Parallel()

	input := ByteTypesStruct{
		ID:         "123-456-789",
		Bytes:      []byte("Hello, World!"),
		BytesArray: [3][]byte{[]byte("Hello"), []byte("World"), []byte("!")},
		JSON:       json.RawMessage(`{"hello": "world"}`),
		JSONArray: [3]json.RawMessage{
			json.RawMessage(`{"hello": "world"}`),
			json.RawMessage(`{"hello": "world"}`),
			json.RawMessage(`{"hello": "world"}`),
		},
		Binary: &Binary{Data: []byte("Hello, World!")},
		BinaryArray: [3]*Binary{
			{Data: []byte("Hello")},
			{Data: []byte("World")},
			{Data: []byte("!")},
		},
	}
	record := core.NewRecord("ByteTypesStruct", input.ID).
		Set("bytes", input.Bytes).
		Set("bytes_array", input.BytesArray).
		Set("json", []byte(input.JSON)).
		Set("json_array", [][]byte{
			[]byte(input.JSONArray[0]),
			[]byte(input.JSONArray[1]),
			[]byte(input.JSONArray[2]),
		}).
		Set("binary", []byte("Hello, World!")).
		Set("binary_array", [][]byte{
			[]byte("Hello"),
			[]byte("World"),
			[]byte("!"),
		})

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type CompositeTypesStruct struct {
	ID             string       `xdb:"id,primary_key"`
	Timestamp      time.Time    `xdb:"timestamp"`
	TimestampArray [3]time.Time `xdb:"timestamp_array"`
}

func TestCompositeTypesStruct(t *testing.T) {
	t.Parallel()

	input := CompositeTypesStruct{
		ID:        "123-456-789",
		Timestamp: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		TimestampArray: [3]time.Time{
			time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC),
			time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		},
	}
	record := core.NewRecord("CompositeTypesStruct", input.ID).
		Set("timestamp", input.Timestamp).
		Set("timestamp_array", input.TimestampArray)

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}

type Post struct {
	ID              string    `xdb:"id,primary_key"`
	Title           string    `xdb:"title"`
	Content         string    `xdb:"content"`
	Author          Author    `xdb:"author"`
	LastComment     Comment   `xdb:"last_comment"`
	CommentCount    int       `xdb:"comment_count"`
	CreatedAt       time.Time `xdb:"created_at"`
	ModerationScore float64   `xdb:"moderation_score"`

	Comments []Comment

	private string
	skipped string `xdb:"-"`
}

type Author struct {
	ID   string `xdb:"id,primary_key"`
	Name string `xdb:"name"`
}

type Comment struct {
	ID              string    `xdb:"id,primary_key"`
	Content         string    `xdb:"content"`
	Author          Author    `xdb:"author"`
	CreatedAt       time.Time `xdb:"created_at"`
	ModerationScore float64   `xdb:"moderation_score"`
}

func TestNestedStruct(t *testing.T) {
	t.Parallel()

	input := Post{
		ID:      "123-456-789",
		Title:   "My First Post",
		Content: "This is my first post",
		Author:  Author{ID: "987-654-321", Name: "John Doe"},
		LastComment: Comment{
			ID:              "111-222-333",
			Content:         "This is my first comment",
			Author:          Author{ID: "999-888-777", Name: "Jane Doe"},
			CreatedAt:       time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
			ModerationScore: 0.5,
		},
		CommentCount:    1,
		CreatedAt:       time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		ModerationScore: 0.5,
		private:         "private",
		skipped:         "skipped",
	}
	record := core.NewRecord("Post", input.ID).
		Set("title", input.Title).
		Set("content", input.Content).
		Set("author.id", input.Author.ID).
		Set("author.name", input.Author.Name).
		Set("last_comment.id", input.LastComment.ID).
		Set("last_comment.content", input.LastComment.Content).
		Set("last_comment.author.id", input.LastComment.Author.ID).
		Set("last_comment.author.name", input.LastComment.Author.Name).
		Set("last_comment.created_at", input.LastComment.CreatedAt).
		Set("last_comment.moderation_score", input.LastComment.ModerationScore).
		Set("comment_count", input.CommentCount).
		Set("created_at", input.CreatedAt).
		Set("moderation_score", input.ModerationScore)

	t.Run("ToRecord", func(t *testing.T) {
		encoded, err := xdbstruct.ToRecord(input)
		require.NoError(t, err)
		require.NotNil(t, encoded)

		tests.AssertEqualRecord(t, record, encoded)
	})
}
