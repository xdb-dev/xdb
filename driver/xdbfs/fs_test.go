package xdbfs_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/driver/xdbfs"
	"github.com/xdb-dev/xdb/tests"
)

func TestFSDriver_Schema(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	driver, err := xdbfs.New(tmpDir)
	require.NoError(t, err)

	tests.TestSchemaReaderWriter(t, driver)
}

func TestFSDriver_Tuples(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	driver, err := xdbfs.New(tmpDir)
	require.NoError(t, err)

	tests.TestTupleReaderWriter(t, driver)
}

func TestFSDriver_Records(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	driver, err := xdbfs.New(tmpDir)
	require.NoError(t, err)

	tests.TestRecordReaderWriter(t, driver)
}
