package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/http"
	"github.com/xdb-dev/xdb/x"
)

type GetTuplesRequest []*Key

type GetTuplesResponse struct {
	Tuples  []*Tuple `json:"tuples"`
	Missing []*Key   `json:"missing"`
}

func GetTuples(store driver.TupleReader) http.EndpointFunc[GetTuplesRequest, GetTuplesResponse] {
	return func(ctx context.Context, req *GetTuplesRequest) (*GetTuplesResponse, error) {
		keys := x.Map(*req, func(key *Key) *core.Key {
			return core.NewKey(key.ID, key.Attr)
		})

		tuples, missing, err := store.GetTuples(ctx, keys)
		if err != nil {
			return nil, err
		}

		resTuples := x.Map(tuples, func(tuple *core.Tuple) *Tuple {
			return &Tuple{
				ID:    tuple.ID(),
				Attr:  tuple.Attr(),
				Value: tuple.Value().Unwrap(),
			}
		})
		resMissing := x.Map(missing, func(key *core.Key) *Key {
			return &Key{ID: key.ID(), Attr: key.Attr()}
		})

		return &GetTuplesResponse{Tuples: resTuples, Missing: resMissing}, nil
	}
}
