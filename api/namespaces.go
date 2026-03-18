package api

import (
	"context"
	"fmt"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// NamespaceService provides namespace operations.
type NamespaceService struct {
	store store.NamespaceReader
}

// NewNamespaceService creates a [NamespaceService] backed by the given [store.NamespaceReader].
func NewNamespaceService(s store.NamespaceReader) *NamespaceService {
	return &NamespaceService{store: s}
}

// GetNamespaceRequest is the request for namespaces.get.
type GetNamespaceRequest struct {
	URI string `json:"uri"`
}

// GetNamespaceResponse is the response for namespaces.get.
type GetNamespaceResponse struct {
	Data *core.NS `json:"data"`
}

// Get retrieves namespace metadata by URI.
func (s *NamespaceService) Get(ctx context.Context, req *GetNamespaceRequest) (*GetNamespaceResponse, error) {
	uri, err := core.ParseURI(req.URI)
	if err != nil {
		return nil, fmt.Errorf("api: namespaces.get: %w", err)
	}

	ns, err := s.store.GetNamespace(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("api: namespaces.get: %w", err)
	}

	return &GetNamespaceResponse{Data: ns}, nil
}

// ListNamespacesRequest is the request for namespaces.list.
type ListNamespacesRequest struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ListNamespacesResponse is the response for namespaces.list.
type ListNamespacesResponse struct {
	Items      []*core.NS `json:"items"`
	NextOffset int        `json:"next_offset,omitempty"`
	Total      int        `json:"total"`
}

// List lists all known namespaces.
func (s *NamespaceService) List(ctx context.Context, req *ListNamespacesRequest) (*ListNamespacesResponse, error) {
	q := &store.ListQuery{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	page, err := s.store.ListNamespaces(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("api: namespaces.list: %w", err)
	}

	return &ListNamespacesResponse{
		Items:      page.Items,
		NextOffset: page.NextOffset,
		Total:      page.Total,
	}, nil
}
