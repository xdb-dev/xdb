package xdbredis_test

import (
	"testing"

	"github.com/xdb-dev/xdb/tests"
)

func TestNamespaces(t *testing.T) {
	tests.NewNamespaceStoreSuite(func() tests.NamespaceStore {
		return newTestStore(t)
	}).Run(t)
}
