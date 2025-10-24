package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
)

type RepoManager interface {
	PutRepo(ctx context.Context, repo *core.Repo) error
}

type RepoAPI struct {
	manager RepoManager
}

func NewRepoAPI(manager RepoManager) *RepoAPI {
	return &RepoAPI{manager: manager}
}

func (a *RepoAPI) PutRepo() EndpointFunc[PutRepoRequest, PutRepoResponse] {
	return func(ctx context.Context, req *PutRepoRequest) (*PutRepoResponse, error) {
		repo, err := core.NewRepo(req.Name)
		if err != nil {
			return nil, err
		}

		repo = repo.WithSchema(req.Schema)

		err = a.manager.PutRepo(ctx, repo)
		if err != nil {
			return nil, err
		}

		return &PutRepoResponse{URI: repo.URI()}, nil
	}
}

type PutRepoRequest struct {
	Name   string       `json:"name"`
	Schema *core.Schema `json:"schema"`
}

type PutRepoResponse struct {
	URI *core.URI `json:"uri"`
}
