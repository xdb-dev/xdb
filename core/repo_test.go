package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/tests"
)

func TestRepo(t *testing.T) {
	t.Run("New Repo", func(t *testing.T) {
		repo, err := core.NewRepo("com.example.posts")
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		assert.Equal(t, core.ModeFlexible, repo.Mode())
		assert.Equal(t, "com.example.posts", repo.Name())
		assert.Equal(t, "com.example.posts", repo.String())
		assert.Equal(t, "xdb://com.example.posts", repo.URI().String())
		assert.Nil(t, repo.Schema())
	})

	t.Run("Valid Repo Names", func(t *testing.T) {
		validNames := []string{
			"posts",
			"com.example.posts",
			"my-repo",
			"my_repo",
			"repo123",
			"a.b.c.d",
			"test-repo_123",
		}

		for _, name := range validNames {
			t.Run(name, func(t *testing.T) {
				repo, err := core.NewRepo(name)
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.Equal(t, name, repo.Name())
			})
		}
	})

	t.Run("Invalid Repo Names", func(t *testing.T) {
		invalidNames := []string{
			"",
			"repo with spaces",
			"repo@invalid",
			"repo#invalid",
			"repo/invalid",
			"repo\\invalid",
			"repo:invalid",
			"repo?invalid",
			"repo&invalid",
			"repo=invalid",
			"repo+invalid",
			"repo*invalid",
			"repo!invalid",
			"repo~invalid",
			"repo`invalid",
			"repo[invalid",
			"repo]invalid",
			"repo{invalid",
			"repo}invalid",
			"repo|invalid",
			"repo<invalid",
			"repo>invalid",
			"repo,invalid",
			"repo;invalid",
		}

		for _, name := range invalidNames {
			t.Run(name, func(t *testing.T) {
				repo, err := core.NewRepo(name)
				assert.Error(t, err)
				assert.Equal(t, core.ErrInvalidRepo, err)
				assert.Nil(t, repo)
			})
		}
	})

	t.Run("Schema Repo", func(t *testing.T) {
		schema := &core.Schema{
			Name:        "com.example.users",
			Description: "User schema",
			Version:     "1.0.0",
			Fields: []*core.FieldSchema{
				{
					Name: "name",
					Type: core.TypeString,
				},
				{
					Name: "age",
					Type: core.TypeInt,
				},
			},
			Required: []string{"name"},
		}
		repo, err := core.NewRepo(schema)
		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.Equal(t, core.ModeStrict, repo.Mode())
		assert.NotNil(t, repo.Schema())
		tests.AssertSchemaEqual(t, schema, repo.Schema())
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Invalid Parameter Type", func(t *testing.T) {
			repo, err := core.NewRepo(123)
			assert.Error(t, err)
			assert.Equal(t, core.ErrInvalidRepo, err)
			assert.Nil(t, repo)
		})
	})
}
