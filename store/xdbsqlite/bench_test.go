package xdbsqlite_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func BenchmarkStore(b *testing.B) {
	tests.NewBenchmarkSuite(func() store.Store {
		dsn := filepath.Join(b.TempDir(), "bench.db") +
			"?_journal=wal&_sync=normal"

		db, err := sql.Open("sqlite3", dsn)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() { db.Close() })

		s, err := xdbsqlite.New(db)
		if err != nil {
			b.Fatal(err)
		}

		return s
	}).Run(b)
}
