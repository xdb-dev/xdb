package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
)

type RepoAPI struct {
	store driver.RepoDriver
}

func NewRepoAPI(store driver.RepoDriver) *RepoAPI {
	return &RepoAPI{store: store}
}

func (a *RepoAPI) MakeRepo() EndpointFunc[MakeRepoRequest, MakeRepoResponse] {
	return func(ctx context.Context, req *MakeRepoRequest) (*MakeRepoResponse, error) {
		repo, err := core.NewRepo(req.Name)
		if err != nil {
			return nil, err
		}

		repo = repo.WithSchema(req.Schema)

		err = a.store.MakeRepo(ctx, repo)
		if err != nil {
			return nil, err
		}

		return &MakeRepoResponse{URI: repo.URI()}, nil
	}
}

type MakeRepoRequest struct {
	Name   string       `json:"name"`
	Schema *core.Schema `json:"schema"`
}

type MakeRepoResponse struct {
	URI *core.URI `json:"uri"`
}
