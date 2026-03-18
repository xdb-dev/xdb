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
func (s *NamespaceService) Get(_ context.Context, _ *GetNamespaceRequest) (*GetNamespaceResponse, error) {
	return nil, fmt.Errorf("api: namespaces.get not implemented")
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
func (s *NamespaceService) List(_ context.Context, _ *ListNamespacesRequest) (*ListNamespacesResponse, error) {
	return nil, fmt.Errorf("api: namespaces.list not implemented")
}
