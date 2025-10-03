package api

import "context"

type PutTuplesRequest []*Tuple

type PutTuplesResponse struct {
}

func PutTuples(ctx context.Context, req *PutTuplesRequest) (*PutTuplesResponse, error) {
	return nil, nil
}
