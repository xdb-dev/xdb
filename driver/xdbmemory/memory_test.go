package xdbmemory

import (
	"testing"

	"github.com/xdb-dev/xdb/tests"
)

func TestMemoryDriver_Tuples(t *testing.T) {
	t.Parallel()
	driver := New()

	tests.TestTupleReaderWriter(t, driver)
}

func TestMemoryDriver_Records(t *testing.T) {
	t.Parallel()

	driver := New()
	tests.TestRecordReaderWriter(t, driver)
}
