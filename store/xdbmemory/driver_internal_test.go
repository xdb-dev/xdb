package xdbmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

func TestApply_EmptyWriteLeavesNoKey(t *testing.T) {
	ctx := context.Background()
	uri := core.MustParseURI("xdb://ns/s/r1")
	record := core.NewRecord("ns", "s", "r1").Set("title", "t")

	tests := []struct {
		name string
		op   store.Op
	}{
		{name: "empty put removes the record", op: store.OpPut},
		{name: "empty create stores no husk", op: store.OpCreate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDriver()
			if tt.op == store.OpPut {
				require.NoError(t, d.Apply(ctx, store.Mutation{
					Path:   uri,
					Op:     store.OpPut,
					Tuples: record.Tuples(),
				}))
			}

			require.NoError(t, d.Apply(ctx, store.Mutation{Path: uri, Op: tt.op}))

			_, ok := d.tuples[recordKey(uri)]
			assert.False(t, ok, "an empty %s must leave no key behind", tt.op)
		})
	}
}
