package x_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/x"
)

func TestMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []int
		want  []string
	}{
		{name: "empty", items: nil, want: []string{}},
		{name: "maps values", items: []int{1, 2, 3}, want: []string{"1", "2", "3"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := x.Map(tc.items, strconv.Itoa)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []int
		want  []int
	}{
		{name: "empty", items: nil, want: []int{}},
		{name: "keeps matching", items: []int{1, 2, 3, 4}, want: []int{2, 4}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := x.Filter(tc.items, func(n int) bool { return n%2 == 0 })
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIndex(t *testing.T) {
	t.Parallel()

	got := x.Index([]string{"a", "bb", "cc"}, func(s string) string {
		return strconv.Itoa(len(s))
	})

	assert.Equal(t, map[string]string{"1": "a", "2": "cc"}, got,
		"later items win on key collision")
}

func TestDiff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{name: "empty", a: nil, b: nil, want: []string{}},
		{name: "a minus b", a: []string{"x", "y", "z"}, b: []string{"y"}, want: []string{"x", "z"}},
		{name: "disjoint", a: []string{"x"}, b: []string{"y"}, want: []string{"x"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := x.Diff(tc.a, tc.b, func(s string) string { return s })
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		lists [][]int
		want  []int
	}{
		{name: "empty", lists: nil, want: nil},
		{name: "concatenates in order", lists: [][]int{{1, 2}, {3}, {}, {4}}, want: []int{1, 2, 3, 4}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := x.Join(tc.lists...)
			assert.Equal(t, tc.want, got)
		})
	}
}
