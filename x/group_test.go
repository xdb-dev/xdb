package x_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

func TestGroupTuples(t *testing.T) {
	t.Parallel()

	t1 := core.NewTuple("app/users/1", "name", "alice")
	t2 := core.NewTuple("app/users/1", "age", int64(30))
	t3 := core.NewTuple("app/users/2", "name", "bob")

	tests := []struct {
		name   string
		tuples []*core.Tuple
		want   map[string][]*core.Tuple
	}{
		{
			name:   "empty",
			tuples: nil,
			want:   map[string][]*core.Tuple{},
		},
		{
			name:   "groups by record path",
			tuples: []*core.Tuple{t1, t2, t3},
			want: map[string][]*core.Tuple{
				"app/users/1": {t1, t2},
				"app/users/2": {t3},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := x.GroupTuples(tc.tuples)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGroupAttrs(t *testing.T) {
	t.Parallel()

	u1 := core.MustParseURI("xdb://app/users/1#name")
	u2 := core.MustParseURI("xdb://app/users/1#age")
	u3 := core.MustParseURI("xdb://app/users/2#name")

	tests := []struct {
		name string
		uris []*core.URI
		want map[string][]string
	}{
		{
			name: "empty",
			uris: nil,
			want: map[string][]string{},
		},
		{
			name: "groups attrs by record path",
			uris: []*core.URI{u1, u2, u3},
			want: map[string][]string{
				"app/users/1": {"name", "age"},
				"app/users/2": {"name"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := x.GroupAttrs(tc.uris)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGroupBy(t *testing.T) {
	t.Parallel()

	got := x.GroupBy([]string{"apple", "avocado", "banana"}, func(s string) string {
		return s[:1]
	})

	require.Len(t, got, 2)
	assert.Equal(t, []string{"apple", "avocado"}, got["a"])
	assert.Equal(t, []string{"banana"}, got["b"])
}
