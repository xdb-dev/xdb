package xdbredis_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/store"
)

func TestStoreImplementsInterfaces(t *testing.T) {
	s := newTestStore(t)

	var _ store.Store = s
	var _ store.HealthChecker = s
}

func TestHealth(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.Health(context.Background()))
}
