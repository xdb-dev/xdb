package xdbsqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
)

func TestMemoryDriver_Repos(t *testing.T) {
	t.Parallel()

	cfg := xdbsqlite.Config{
		Dir:  t.TempDir(),
		Name: "test.db",
	}
	driver, err := xdbsqlite.New(cfg)
	require.NoError(t, err)

	tests.TestRepoReaderWriter(t, driver)
}

func TestMemoryDriver_KVRepos(t *testing.T) {
	t.Parallel()

	cfg := xdbsqlite.Config{
		Dir:  t.TempDir(),
		Name: "test.db",
	}
	driver, err := xdbsqlite.New(cfg)
	require.NoError(t, err)

	repo, err := core.NewRepo("testkv")
	require.NoError(t, err)

	err = driver.MakeRepo(context.Background(), repo)
	require.NoError(t, err)

	got, err := driver.GetRepo(context.Background(), repo.Name())
	require.NoError(t, err)
	require.Equal(t, repo, got)
}
