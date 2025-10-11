package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/x"
)

type PutTuplesRequest []*Tuple

type PutTuplesResponse struct{}

func PutTuples(store driver.TupleWriter) EndpointFunc[PutTuplesRequest, PutTuplesResponse] {
	return func(ctx context.Context, req *PutTuplesRequest) (*PutTuplesResponse, error) {
		tuples := x.Map(*req, func(tuple *Tuple) *core.Tuple {
			return core.NewTuple(tuple.ID, tuple.Attr, tuple.Value)
		})

		err := store.PutTuples(ctx, tuples)
		if err != nil {
			return nil, err
		}

		return &PutTuplesResponse{}, nil
	}
}
