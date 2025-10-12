package api

import "context"

type Tuple struct {
	ID    []string `json:"id"`
	Attr  []string `json:"attr"`
	Value any      `json:"value"`
}

type Key struct {
	ID   []string `json:"id"`
	Attr []string `json:"attr"`
}

type EndpointFunc[Req any, Res any] func(ctx context.Context, req *Req) (*Res, error)
