package x_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

func TestURIs(t *testing.T) {
	t.Parallel()

	t.Run("tuples", func(t *testing.T) {
		t.Parallel()

		tuples := []*core.Tuple{
			core.NewTuple("app/users/1", "name", "alice"),
			core.NewTuple("app/users/2", "name", "bob"),
		}

		got := x.URIs(tuples)

		assert.Equal(t, []*core.URI{
			core.MustParseURI("xdb://app/users/1#name"),
			core.MustParseURI("xdb://app/users/2#name"),
		}, got)
	})

	t.Run("records", func(t *testing.T) {
		t.Parallel()

		records := []*core.Record{
			core.NewRecord("app", "users", "1"),
		}

		got := x.URIs(records)

		assert.Equal(t, []*core.URI{
			core.MustParseURI("xdb://app/users/1"),
		}, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		got := x.URIs([]*core.Tuple{})
		assert.Empty(t, got)
	})
}
