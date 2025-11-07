package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestID(t *testing.T) {
	t.Parallel()

	t.Run("IsEmpty", func(t *testing.T) {
		emptyID := core.NewID()
		assert.True(t, emptyID.IsEmpty())

		nonEmptyID := core.NewID("123")
		assert.False(t, nonEmptyID.IsEmpty())
	})

	t.Run("IsValid", func(t *testing.T) {
		testcases := []struct {
			name  string
			id    core.ID
			valid bool
		}{
			{
				name:  "Empty ID",
				id:    core.NewID(),
				valid: false,
			},
			{
				name:  "Single component",
				id:    core.NewID("123"),
				valid: true,
			},
			{
				name:  "Multiple components",
				id:    core.NewID("org", "dept", "user"),
				valid: true,
			},
			{
				name:  "Empty string component",
				id:    core.ID{""},
				valid: false,
			},
			{
				name:  "Empty string in middle",
				id:    core.ID{"a", "", "c"},
				valid: false,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.valid, tc.id.IsValid())
			})
		}
	})

	t.Run("Equals", func(t *testing.T) {
		id1 := core.NewID("123")
		id2 := core.NewID("123")
		id3 := core.NewID("456")
		id4 := core.NewID("123", "456")

		assert.True(t, id1.Equals(id2))
		assert.False(t, id1.Equals(id3))
		assert.False(t, id1.Equals(id4))
	})

	t.Run("String", func(t *testing.T) {
		id := core.NewID("org", "dept", "user")
		assert.Equal(t, "org/dept/user", id.String())

		singleID := core.NewID("123")
		assert.Equal(t, "123", singleID.String())
	})
}

func TestAttr(t *testing.T) {
	t.Parallel()

	t.Run("IsEmpty", func(t *testing.T) {
		emptyAttr := core.NewAttr()
		assert.True(t, emptyAttr.IsEmpty())

		nonEmptyAttr := core.NewAttr("name")
		assert.False(t, nonEmptyAttr.IsEmpty())
	})

	t.Run("IsValid", func(t *testing.T) {
		testcases := []struct {
			name  string
			attr  core.Attr
			valid bool
		}{
			{
				name:  "Empty Attr",
				attr:  core.NewAttr(),
				valid: false,
			},
			{
				name:  "Single component",
				attr:  core.NewAttr("name"),
				valid: true,
			},
			{
				name:  "Multiple components",
				attr:  core.NewAttr("profile", "name"),
				valid: true,
			},
			{
				name:  "Empty string component",
				attr:  core.Attr{""},
				valid: false,
			},
			{
				name:  "Empty string in middle",
				attr:  core.Attr{"a", "", "c"},
				valid: false,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.valid, tc.attr.IsValid())
			})
		}
	})

	t.Run("Equals", func(t *testing.T) {
		attr1 := core.NewAttr("name")
		attr2 := core.NewAttr("name")
		attr3 := core.NewAttr("email")
		attr4 := core.NewAttr("profile", "name")

		assert.True(t, attr1.Equals(attr2))
		assert.False(t, attr1.Equals(attr3))
		assert.False(t, attr1.Equals(attr4))
	})

	t.Run("String", func(t *testing.T) {
		attr := core.NewAttr("profile", "email")
		assert.Equal(t, "profile.email", attr.String())

		singleAttr := core.NewAttr("name")
		assert.Equal(t, "name", singleAttr.String())
	})

	t.Run("Dot-Splitting Behavior", func(t *testing.T) {
		// Single string with dots should be split
		attr1 := core.NewAttr("profile.email")
		assert.Equal(t, 2, len(attr1))
		assert.Equal(t, "profile", attr1[0])
		assert.Equal(t, "email", attr1[1])
		assert.Equal(t, "profile.email", attr1.String())

		// Multiple strings with dots in first arg should still split
		attr2 := core.NewAttr("a.b.c")
		assert.Equal(t, 3, len(attr2))
		assert.Equal(t, "a", attr2[0])
		assert.Equal(t, "b", attr2[1])
		assert.Equal(t, "c", attr2[2])

		// Multiple args should NOT split
		attr3 := core.NewAttr("profile", "email")
		assert.Equal(t, 2, len(attr3))
		assert.Equal(t, "profile", attr3[0])
		assert.Equal(t, "email", attr3[1])

		// Single string without dots should not split
		attr4 := core.NewAttr("name")
		assert.Equal(t, 1, len(attr4))
		assert.Equal(t, "name", attr4[0])

		// Edge case: empty string
		attr5 := core.NewAttr("")
		assert.Equal(t, 1, len(attr5))
		assert.Equal(t, "", attr5[0])

		// Edge case: just dots
		attr6 := core.NewAttr("...")
		assert.Equal(t, 4, len(attr6))
		for _, component := range attr6 {
			assert.Equal(t, "", component)
		}
	})
}

func TestTuple(t *testing.T) {
	t.Parallel()

	t.Run("NewTuple", func(t *testing.T) {
		tuple := core.NewTuple(
			"com.example.posts",
			"123",
			"title",
			"Hello, World!",
		)

		assert.NotNil(t, tuple)
		assert.Equal(t, "com.example.posts", tuple.Repo())
		assert.Equal(t, "title", tuple.Attr().String())
		assert.Equal(t, "123", tuple.ID().String())
		assert.Equal(t, "Hello, World!", tuple.Value().ToString())
		assert.Equal(t, "xdb://com.example.posts/123#title", tuple.URI().String())
		assert.Equal(t, "Tuple(com.example.posts, 123, title, Value(STRING, Hello, World!))", tuple.GoString())
	})

	t.Run("Hierarchy ID", func(t *testing.T) {
		id := core.NewID("org-123", "user-456")
		tuple1 := core.NewTuple("com.example.users", id, "name", "John")
		assert.Equal(t, "com.example.users", tuple1.Repo())
		assert.Equal(t, "org-123/user-456", tuple1.ID().String())
		assert.Equal(t, "name", tuple1.Attr().String())
	})

	t.Run("Nested Attr", func(t *testing.T) {
		attr := core.NewAttr("profile", "name")
		tuple := core.NewTuple("com.example.users", "123", attr, "John")
		assert.Equal(t, "profile.name", tuple.Attr().String())
	})

	t.Run("Nil Value", func(t *testing.T) {
		tuple := core.NewTuple("com.example.users", "123", "name", nil)
		assert.NotNil(t, tuple)
		assert.True(t, tuple.IsNil())
	})

	t.Run("Invalid ID Type", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewTuple("com.example.users", 123, "name", "John") // Invalid ID type
		})
	})

	t.Run("Invalid Attr Type", func(t *testing.T) {
		assert.Panics(t, func() {
			core.NewTuple("com.example.users", "123", 123, "John") // Invalid Attr type
		})
	})

	t.Run("Empty ID", func(t *testing.T) {
		tuple := core.NewTuple("com.example.users", "", "name", "John")
		assert.Equal(t, "", tuple.ID().String())
	})

	t.Run("Empty Attr", func(t *testing.T) {
		tuple := core.NewTuple("com.example.users", "123", "", "John")
		assert.Equal(t, "", tuple.Attr().String())
	})
}
