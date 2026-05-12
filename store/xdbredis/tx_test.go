package xdbredis_test

import (
	"testing"

	"github.com/xdb-dev/xdb/tests"
)

func TestBatch(t *testing.T) {
	tests.NewBatchSuite(func() tests.BatchStore {
		return newTestStore(t)
	}).Run(t)
}
