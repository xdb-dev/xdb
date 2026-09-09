package x_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xdb-dev/xdb/x"
)

func TestMap(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []string
	}{
		{"empty", []int{}, []string{}},
		{"nil", nil, []string{}},
		{"values", []int{1, 2, 3}, []string{"1", "2", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, x.Map(tt.items, strconv.Itoa))
		})
	}
}

func TestIndex(t *testing.T) {
	t.Run("keys by fn", func(t *testing.T) {
		got := x.Index([]string{"a", "bb", "cc"}, func(s string) string {
			return strconv.Itoa(len(s))
		})

		assert.Equal(t, map[string]string{"1": "a", "2": "cc"}, got)
	})

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, x.Index([]string{}, func(s string) string { return s }))
	})
}
