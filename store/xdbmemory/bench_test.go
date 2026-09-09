package xdbmemory_test

import (
	"testing"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/storetest"
)

func BenchmarkStore(b *testing.B) {
	storetest.NewBenchmarkSuite(func() store.Store {
		return store.New(xdbmemory.NewDriver())
	}).Run(b)
}
