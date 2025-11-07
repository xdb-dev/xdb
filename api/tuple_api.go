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
			return core.NewTuple(tuple.Repo, tuple.ID, tuple.Attr, tuple.Value)
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
		uris := x.Map(*req, func(uri *URI) *core.URI {
			return core.NewURI(uri.Repo, uri.ID, uri.Attr)
		})

		tuples, missing, err := s.store.GetTuples(ctx, uris)
		if err != nil {
			return nil, err
		}

		resTuples := x.Map(tuples, func(tuple *core.Tuple) *Tuple {
			return &Tuple{
				Repo:  tuple.Repo(),
				ID:    tuple.ID(),
				Attr:  tuple.Attr(),
				Value: tuple.Value().Unwrap(),
			}
		})
		resMissing := x.Map(missing, func(uri *core.URI) *URI {
			return &URI{Repo: uri.Repo(), ID: uri.ID(), Attr: uri.Attr()}
		})

		return &GetTuplesResponse{Tuples: resTuples, Missing: resMissing}, nil
	}
}

func (s *TupleAPI) DeleteTuples() EndpointFunc[DeleteTuplesRequest, DeleteTuplesResponse] {
	return func(ctx context.Context, req *DeleteTuplesRequest) (*DeleteTuplesResponse, error) {
		uris := x.Map(*req, func(uri *URI) *core.URI {
			return core.NewURI(uri.Repo, uri.ID, uri.Attr)
		})

		err := s.store.DeleteTuples(ctx, uris)
		if err != nil {
			return nil, err
		}

		return &DeleteTuplesResponse{}, nil
	}
}

type PutTuplesRequest []*Tuple

type PutTuplesResponse struct{}

type GetTuplesRequest []*URI

type GetTuplesResponse struct {
	Tuples  []*Tuple
	Missing []*URI
}

type DeleteTuplesRequest []*URI

type DeleteTuplesResponse struct{}
