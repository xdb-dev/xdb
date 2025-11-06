package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
)

func TestRepo(t *testing.T) {
	t.Run("Getters", func(t *testing.T) {
		repo, err := core.NewRepo("com.example.posts")
		assert.NoError(t, err)
		assert.NotNil(t, repo)

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

	t.Run("WithSchema", func(t *testing.T) {
		repo, err := core.NewRepo("com.example.users")
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		schema := &core.Schema{
			Name:        "User",
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

		repo = repo.WithSchema(schema)
		assert.NotNil(t, repo.Schema())
		assert.Equal(t, "User", repo.Schema().Name)
		assert.Equal(t, "1.0.0", repo.Schema().Version)
		assert.Equal(t, 2, len(repo.Schema().Fields))
	})

	t.Run("Edge Cases", func(t *testing.T) {
		t.Run("Nil Schema", func(t *testing.T) {
			repo, err := core.NewRepo("test.repo")
			assert.NoError(t, err)

			repo = repo.WithSchema(nil)
			assert.Nil(t, repo.Schema())
		})

		t.Run("Multiple WithSchema Calls", func(t *testing.T) {
			repo, err := core.NewRepo("test.repo")
			assert.NoError(t, err)

			schema1 := &core.Schema{Name: "Schema1"}
			schema2 := &core.Schema{Name: "Schema2"}

			repo = repo.WithSchema(schema1)
			assert.Equal(t, "Schema1", repo.Schema().Name)

			assert.Panics(t, func() {
				_ = repo.WithSchema(schema2)
			})
		})

		t.Run("URI Format", func(t *testing.T) {
			repo, err := core.NewRepo("com.example.posts")
			assert.NoError(t, err)

			uri := repo.URI()
			assert.NotNil(t, uri)
			assert.Equal(t, "com.example.posts", uri.Repo())
			assert.Equal(t, "xdb://com.example.posts", uri.String())
		})
	})
}
