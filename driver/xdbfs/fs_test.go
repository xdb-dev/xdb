package xdbfs_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/driver/xdbfs"
	"github.com/xdb-dev/xdb/tests"
)

func TestFSRepoDriver(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "xdbfs")

	driver, err := xdbfs.New(dir)
	require.NoError(t, err)

	tests.TestRepoReaderWriter(t, driver)
}
