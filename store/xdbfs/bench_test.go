package xdbfs_test

import (
	"testing"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbfs"
	"github.com/xdb-dev/xdb/tests"
)

func BenchmarkStore(b *testing.B) {
	tests.NewBenchmarkSuite(func() store.Store {
		d, err := xdbfs.NewDriver(b.TempDir(), xdbfs.Options{})
		if err != nil {
			b.Fatal(err)
		}
		return store.New(d)
	}).Run(b)
}
