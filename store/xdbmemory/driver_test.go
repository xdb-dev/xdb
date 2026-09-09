package xdbmemory_test

import (
	"testing"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/storetest"
)

func TestDriverSuite(t *testing.T) {
	suite := storetest.NewDriverSuite(func() store.Driver {
		return xdbmemory.NewDriver()
	})
	suite.Run(t)
}

func TestQuerySuite(t *testing.T) {
	storetest.NewQuerySuite(func() store.Driver {
		return xdbmemory.NewDriver()
	}).Run(t)
}
