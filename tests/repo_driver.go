package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
)

type repoReaderWriter interface {
	driver.RepoReader
	driver.RepoWriter
}

func TestRepoReaderWriter(t *testing.T, rw repoReaderWriter) {
	t.Helper()

	ctx := context.Background()
	repo := FakeRepo()

	t.Run("MakeRepo", func(t *testing.T) {
		err := rw.MakeRepo(ctx, repo)
		require.NoError(t, err)
	})

	t.Run("GetRepo", func(t *testing.T) {
		got, err := rw.GetRepo(ctx, repo.Name())
		require.NoError(t, err)
		AssertEqualRepo(t, repo, got)
	})

	t.Run("ListRepos", func(t *testing.T) {
		got, err := rw.ListRepos(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, got)
		// At least the repo we created should be in the list
		found := false
		for _, r := range got {
			if r.Name() == repo.Name() {
				found = true
				AssertEqualRepo(t, repo, r)
				break
			}
		}
		require.True(t, found, "created repo not found in list")
	})

	t.Run("DeleteRepo", func(t *testing.T) {
		err := rw.DeleteRepo(ctx, repo.Name())
		require.NoError(t, err)
	})

	t.Run("GetRepo", func(t *testing.T) {
		got, err := rw.GetRepo(ctx, repo.Name())
		require.Error(t, err)
		require.ErrorIs(t, err, driver.ErrNotFound)
		require.Nil(t, got)
	})
}

func AssertEqualRepos(t *testing.T, expected, actual []*core.Repo) {
	t.Helper()

	require.Equal(t, len(expected), len(actual), "repo lists have different lengths")

	for i, expected := range expected {
		actual := actual[i]
		AssertEqualRepo(t, expected, actual)
	}
}

func AssertEqualRepo(t *testing.T, expected, actual *core.Repo) {
	t.Helper()

	require.Equal(t, expected.Name(), actual.Name(), "repo names are different")
	AssertSchemaDefEqual(t, expected.Schema(), actual.Schema())
}
