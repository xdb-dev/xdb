package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func userDef() *schema.Def {
	return &schema.Def{
		URI:      core.MustParseURI("xdb://app/users"),
		Mode:     schema.ModeStrict,
		Revision: 3,
		Fields: map[string]schema.Field{
			"name": {Type: core.TypeString, Required: true},
		},
	}
}

func TestStampSystemFields(t *testing.T) {
	t.Parallel()

	t.Run("adds version and updated, leaves id virtual", func(t *testing.T) {
		stamped := schema.StampSystemFields(userDef())

		require.Contains(t, stamped.Fields, schema.FieldVersion)
		require.Contains(t, stamped.Fields, schema.FieldUpdated)
		assert.NotContains(t, stamped.Fields, schema.FieldID)

		assert.Equal(t, core.TypeInt, stamped.Fields[schema.FieldVersion].Type)
		assert.Equal(t, core.TypeTime, stamped.Fields[schema.FieldUpdated].Type)
	})

	t.Run("system fields are never required", func(t *testing.T) {
		stamped := schema.StampSystemFields(userDef())

		assert.False(t, stamped.Fields[schema.FieldVersion].Required)
		assert.False(t, stamped.Fields[schema.FieldUpdated].Required)
	})

	t.Run("preserves user fields and metadata", func(t *testing.T) {
		stamped := schema.StampSystemFields(userDef())

		assert.Equal(t, core.TypeString, stamped.Fields["name"].Type)
		assert.True(t, stamped.Fields["name"].Required)
		assert.Equal(t, schema.ModeStrict, stamped.Mode)
		assert.Equal(t, int64(3), stamped.Revision)
		assert.Equal(t, "xdb://app/users", stamped.URI.String())
	})

	t.Run("does not mutate the receiver", func(t *testing.T) {
		def := userDef()
		schema.StampSystemFields(def)

		assert.NotContains(t, def.Fields, schema.FieldVersion)
	})

	t.Run("is idempotent", func(t *testing.T) {
		once := schema.StampSystemFields(userDef())
		twice := schema.StampSystemFields(once)

		assert.Equal(t, once.Fields, twice.Fields)
	})

	t.Run("handles a nil field map", func(t *testing.T) {
		def := &schema.Def{
			URI:  core.MustParseURI("xdb://app/users"),
			Mode: schema.ModeFlexible,
		}
		stamped := schema.StampSystemFields(def)

		assert.Contains(t, stamped.Fields, schema.FieldVersion)
		assert.Contains(t, stamped.Fields, schema.FieldUpdated)
	})
}

func TestStripSystemFields(t *testing.T) {
	t.Parallel()

	t.Run("removes every system field", func(t *testing.T) {
		def := userDef()
		def.Fields[schema.FieldVersion] = schema.Field{Type: core.TypeInt}
		def.Fields[schema.FieldUpdated] = schema.Field{Type: core.TypeTime}
		def.Fields[schema.FieldID] = schema.Field{Type: core.TypeString}

		stripped := schema.StripSystemFields(def)

		assert.NotContains(t, stripped.Fields, schema.FieldVersion)
		assert.NotContains(t, stripped.Fields, schema.FieldUpdated)
		assert.NotContains(t, stripped.Fields, schema.FieldID)
		assert.Contains(t, stripped.Fields, "name")
	})

	t.Run("does not mutate the receiver", func(t *testing.T) {
		def := schema.StampSystemFields(userDef())
		schema.StripSystemFields(def)

		assert.Contains(t, def.Fields, schema.FieldVersion)
	})

	t.Run("round-trips with stamp", func(t *testing.T) {
		def := userDef()
		back := schema.StripSystemFields(schema.StampSystemFields(def))

		assert.Equal(t, def.Fields, back.Fields)
	})

	t.Run("a stripped def validates", func(t *testing.T) {
		stamped := schema.StampSystemFields(userDef())

		require.Error(t, stamped.Validate())
		assert.NoError(t, schema.StripSystemFields(stamped).Validate())
	})
}

func TestHasSystemFields(t *testing.T) {
	t.Parallel()

	assert.True(t, schema.HasSystemFields(schema.StampSystemFields(userDef())))
	assert.False(t, schema.HasSystemFields(userDef()))

	t.Run("partial stamping counts as missing", func(t *testing.T) {
		def := userDef()
		def.Fields[schema.FieldVersion] = schema.Field{Type: core.TypeInt}

		assert.False(t, schema.HasSystemFields(def))
	})
}
