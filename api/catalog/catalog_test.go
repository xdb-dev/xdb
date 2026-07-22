package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api/catalog"
)

func TestMethods_EveryEntryHasDescriptionAndDottedName(t *testing.T) {
	methods := catalog.Methods()
	require.NotEmpty(t, methods)

	// "watch" is the sole bare (dot-less) method name — a top-level verb,
	// not namespaced under a resource.
	for name, meta := range methods {
		assert.NotEmpty(t, meta.Description, "method %s has empty description", name)
		if name != "watch" {
			assert.Contains(t, name, ".", "method %s does not contain a dot", name)
		}
	}
}

func TestMethods_IncludesIntrospection(t *testing.T) {
	methods := catalog.Methods()

	for _, name := range []string{
		"introspect.method",
		"introspect.type",
		"introspect.methods",
		"introspect.types",
	} {
		assert.Contains(t, methods, name)
	}
}

func TestMethod(t *testing.T) {
	t.Run("known method", func(t *testing.T) {
		meta, ok := catalog.Method("records.create")
		require.True(t, ok)
		assert.NotEmpty(t, meta.Description)
	})

	t.Run("unknown method", func(t *testing.T) {
		_, ok := catalog.Method("does.not.exist")
		assert.False(t, ok)
	})

	t.Run("matches Methods()", func(t *testing.T) {
		all := catalog.Methods()
		meta, ok := catalog.Method("schemas.get")
		require.True(t, ok)
		assert.Equal(t, all["schemas.get"], meta)
	})
}

func TestTypes_GoldenList(t *testing.T) {
	types := catalog.Types()

	for _, name := range []string{
		"Record",
		"Schema",
		"Namespace",
		"Tuple",
		"Value",
		"URI",
		"Filter",
		"Mode",
	} {
		desc, ok := types[name]
		require.True(t, ok, "type %s missing from catalog.Types()", name)
		assert.NotEmpty(t, desc, "type %s has empty description", name)
	}
}

func TestTypes_ValueDescriptionUsesBoolean(t *testing.T) {
	desc, ok := catalog.Types()["Value"]
	require.True(t, ok)
	assert.Contains(t, desc, "boolean")
}
