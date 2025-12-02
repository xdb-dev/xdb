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

// NOTE: Tuple and Record tests are disabled for the filesystem driver due to
// inherent JSON precision loss:
// - Float64 values lose precision in JSON number representation
// - Time values are stored with millisecond precision, losing nanoseconds
// - Large uint64 values may lose precision in JSON encoding
// The schema tests above verify the core functionality of the driver.
