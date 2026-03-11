package xdbredis_test

import (
	"testing"

	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/tests"
)

func TestRecords(t *testing.T) {
	tests.NewRecordStoreSuite(func() store.RecordStore {
		return newTestStore(t)
	}).Run(t)
}
