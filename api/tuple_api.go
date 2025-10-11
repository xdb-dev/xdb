package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/x"
)

type TupleStorer interface {
	driver.TupleReader
	driver.TupleWriter
}

type TupleAPI struct {
	store TupleStorer
}

func NewTupleAPI(store TupleStorer) *TupleAPI {
	return &TupleAPI{store: store}
}

func (s *TupleAPI) PutTuples() EndpointFunc[PutTuplesRequest, PutTuplesResponse] {
	return func(ctx context.Context, req *PutTuplesRequest) (*PutTuplesResponse, error) {
		tuples := x.Map(*req, func(tuple *Tuple) *core.Tuple {
			return core.NewTuple(tuple.ID, tuple.Attr, tuple.Value)
		})

		err := s.store.PutTuples(ctx, tuples)
		if err != nil {
			return nil, err
		}

		return &PutTuplesResponse{}, nil
	}
}

func (s *TupleAPI) GetTuples() EndpointFunc[GetTuplesRequest, GetTuplesResponse] {
	return func(ctx context.Context, req *GetTuplesRequest) (*GetTuplesResponse, error) {
		keys := x.Map(*req, func(key *Key) *core.Key {
			return core.NewKey(key.ID, key.Attr)
		})

		tuples, missing, err := s.store.GetTuples(ctx, keys)
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

func (s *TupleAPI) DeleteTuples() EndpointFunc[DeleteTuplesRequest, DeleteTuplesResponse] {
	return func(ctx context.Context, req *DeleteTuplesRequest) (*DeleteTuplesResponse, error) {
		keys := x.Map(*req, func(key *Key) *core.Key {
			return core.NewKey(key.ID, key.Attr)
		})

		err := s.store.DeleteTuples(ctx, keys)
		if err != nil {
			return nil, err
		}

		return &DeleteTuplesResponse{}, nil
	}
}

type PutTuplesRequest []*Tuple

type PutTuplesResponse struct{}

type GetTuplesRequest []*Key

type GetTuplesResponse struct {
	Tuples  []*Tuple
	Missing []*Key
}

type DeleteTuplesRequest []*Key

type DeleteTuplesResponse struct{}
