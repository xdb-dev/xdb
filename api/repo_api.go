package api

import (
	"context"

	"github.com/xdb-dev/xdb/core"
)

type RepoManager interface {
	CreateRepo(ctx context.Context, repo *core.Repo) error
	CreateCollection(ctx context.Context, collection *core.Collection) error
}

type RepoAPI struct {
	manager RepoManager
}

func NewRepoAPI(manager RepoManager) *RepoAPI {
	return &RepoAPI{manager: manager}
}

func (a *RepoAPI) CreateRepo() EndpointFunc[CreateRepoRequest, CreateRepoResponse] {
	return func(ctx context.Context, req *CreateRepoRequest) (*CreateRepoResponse, error) {
		repo, err := core.NewRepo(req.Name)
		if err != nil {
			return nil, err
		}

		err = a.manager.CreateRepo(ctx, repo)
		if err != nil {
			return nil, err
		}

		return &CreateRepoResponse{URI: repo.URI()}, nil
	}
}

func (a *RepoAPI) CreateCollection() EndpointFunc[CreateCollectionRequest, CreateCollectionResponse] {
	return func(ctx context.Context, req *CreateCollectionRequest) (*CreateCollectionResponse, error) {
		collection := core.Collection(*req)

		err := a.manager.CreateCollection(ctx, &collection)
		if err != nil {
			return nil, err
		}

		return &CreateCollectionResponse{URI: collection.URI()}, nil
	}
}

type CreateRepoRequest struct {
	Name string `json:"name"`
}

type CreateRepoResponse struct {
	URI *core.URI `json:"uri"`
}

type CreateCollectionRequest core.Collection

type CreateCollectionResponse struct {
	URI *core.URI `json:"uri"`
}
