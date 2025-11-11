package api

import (
	"context"
	"errors"

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
		var repo *core.Repo
		var err error

		if req.Schema == nil {
			repo, err = core.NewRepo(req.Name)
		} else {
			repo, err = core.NewRepo(req.Schema)
		}

		if err != nil {
			return nil, err
		}

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

func (req *MakeRepoRequest) Validate() error {
	if req.Name == "" && req.Schema == nil {
		return errors.New("name or schema is required")
	}
	return nil
}

type MakeRepoResponse struct {
	URI *core.URI `json:"uri"`
}
