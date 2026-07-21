package xdbfs_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbfs"
	"github.com/xdb-dev/xdb/tests"
)

func TestDriverSuite(t *testing.T) {
	suite := tests.NewDriverSuite(func() store.Driver {
		d, err := xdbfs.NewDriver(t.TempDir(), xdbfs.Options{})
		require.NoError(t, err)
		return d
	})
	suite.Run(t)
}

func TestQuerySuite(t *testing.T) {
	tests.NewQuerySuite(func() store.Driver {
		d, err := xdbfs.NewDriver(t.TempDir(), xdbfs.Options{})
		require.NoError(t, err)
		return d
	}).Run(t)
}
