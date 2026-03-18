package xdbmemory_test

import (
	"testing"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
	"github.com/xdb-dev/xdb/tests"
)

func BenchmarkStore(b *testing.B) {
	tests.NewBenchmarkSuite(func() store.Store {
		return xdbmemory.New()
	}).Run(b)
}
