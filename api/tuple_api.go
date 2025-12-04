package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/x"
)

type TupleAPI struct {
	store driver.TupleDriver
}

func NewTupleAPI(store driver.TupleDriver) *TupleAPI {
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
		uris := x.Map(*req, func(uriStr string) *core.URI {
			uri, err := core.ParseURI(uriStr)
			if err != nil {
				panic(err)
			}
			return uri
		})

		tuples, missing, err := s.store.GetTuples(ctx, uris)
		if err != nil {
			return nil, err
		}

		resTuples := x.Map(tuples, func(tuple *core.Tuple) *Tuple {
			return &Tuple{
				ID:    tuple.URI().ID().String(),
				Attr:  tuple.Attr().String(),
				Value: tuple.Value().Unwrap(),
			}
		})
		resMissing := x.Map(missing, func(uri *core.URI) string {
			return uri.String()
		})

		return &GetTuplesResponse{Tuples: resTuples, Missing: resMissing}, nil
	}
}

func (s *TupleAPI) DeleteTuples() EndpointFunc[DeleteTuplesRequest, DeleteTuplesResponse] {
	return func(ctx context.Context, req *DeleteTuplesRequest) (*DeleteTuplesResponse, error) {
		uris := x.Map(*req, func(uriStr string) *core.URI {
			uri, err := core.ParseURI(uriStr)
			if err != nil {
				panic(err)
			}
			return uri
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

type GetTuplesRequest []string

type GetTuplesResponse struct {
	Tuples  []*Tuple `json:"tuples"`
	Missing []string `json:"missing"`
}

type DeleteTuplesRequest []string

type DeleteTuplesResponse struct{}
