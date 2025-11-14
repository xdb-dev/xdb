package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/tests"
)

func TestSchemaDef(t *testing.T) {
	t.Parallel()

	schema := &core.SchemaDef{
		Name:        "users",
		Description: "User schema",
		Version:     "1.0.0",
		Mode:        core.ModeStrict,
		Fields: []*core.FieldDef{
			{Name: "name", Type: core.TypeString},
		},
	}

	t.Run("Clone", func(t *testing.T) {
		clone := schema.Clone()
		tests.AssertSchemaDefEqual(t, schema, clone)
	})

	t.Run("GetField", func(t *testing.T) {
		got := schema.GetField("name")
		assert.NotNil(t, got)
		assert.Equal(t, "name", got.Name)
		assert.Equal(t, core.TypeString, got.Type)
	})

	t.Run("GetField not found", func(t *testing.T) {
		got := schema.GetField("not-found")
		assert.Nil(t, got)
	})
}
