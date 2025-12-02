package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/tests"
)

func TestDef(t *testing.T) {
	t.Parallel()

	s := &schema.Def{
		Name:        "users",
		Description: "User schema",
		Version:     "1.0.0",
		Mode:        schema.ModeStrict,
		Fields: []*schema.FieldDef{
			{Name: "name", Type: core.TypeString},
		},
	}

	t.Run("Clone", func(t *testing.T) {
		clone := s.Clone()
		tests.AssertDefEqual(t, s, clone)
	})

	t.Run("GetField", func(t *testing.T) {
		got := s.GetField("name")
		assert.NotNil(t, got)
		assert.Equal(t, "name", got.Name)
		assert.Equal(t, core.TypeString, got.Type)
	})

	t.Run("GetField not found", func(t *testing.T) {
		got := s.GetField("not-found")
		assert.Nil(t, got)
	})
}
