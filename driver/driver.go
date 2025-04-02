package driver

import (
	"context"

	"github.com/xdb-dev/xdb/types"
)

// TupleReader is an interface for reading tuples.
type TupleReader interface {
	GetTuples(ctx context.Context, keys []*types.Key) ([]*types.Tuple, []*types.Key, error)
}

// TupleWriter is an interface for writing & deleting tuples.
type TupleWriter interface {
	PutTuples(ctx context.Context, tuples []*types.Tuple) error
	DeleteTuples(ctx context.Context, keys []*types.Key) error
}
