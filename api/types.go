package api

import "context"

type Tuple struct {
	Repo  string   `json:"repo"`
	ID    []string `json:"id"`
	Attr  []string `json:"attr"`
	Value any      `json:"value"`
}

type URI struct {
	Repo string   `json:"repo"`
	ID   []string `json:"id"`
	Attr []string `json:"attr"`
}

type EndpointFunc[Req any, Res any] func(ctx context.Context, req *Req) (*Res, error)
