package api

import (
	"context"
	"fmt"
	"sort"

	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store"
)

// namespaceStore is the slice of [store.Store] the namespace service
// needs: namespace reads plus schema listing for tree discovery.
type namespaceStore interface {
	store.NamespaceReader

	// ListSchemas lists schemas matching the given query.
	ListSchemas(ctx context.Context, q *store.Query) (*store.Page[*schema.Def], error)
}

// NamespaceService provides namespace operations.
type NamespaceService struct {
	store namespaceStore
}

// NewNamespaceService creates a [NamespaceService] backed by the given store.
func NewNamespaceService(s namespaceStore) *NamespaceService {
	return &NamespaceService{store: s}
}

// GetNamespaceRequest is the request for namespaces.get.
type GetNamespaceRequest struct {
	URI string `json:"uri"`
}

// GetNamespaceResponse is the response for namespaces.get.
type GetNamespaceResponse struct {
	Data         string   `json:"data"`
	Schemas      []string `json:"schemas"`
	TotalSchemas int      `json:"total_schemas"`
}

// Get retrieves namespace metadata by URI. Schemas holds every schema URI
// in the namespace, sorted, so agents can walk the tree. TotalSchemas
// reports the same count. The schemas are read a page at a time, because
// the store caps a single list call at [store.MaxLimit].
func (s *NamespaceService) Get(ctx context.Context, req *GetNamespaceRequest) (*GetNamespaceResponse, error) {
	uri, err := parseURI(req.URI, "namespaces.get", 1, 1, false)
	if err != nil {
		return nil, fmt.Errorf("api: namespaces.get: %w", err)
	}

	ns, err := s.store.GetNamespace(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("api: namespaces.get: %w", err)
	}

	var (
		schemas []string
		total   int
		offset  int
	)

	for {
		page, err := s.store.ListSchemas(ctx, &store.Query{
			URI:    uri,
			Limit:  store.MaxLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("api: namespaces.get: %w", err)
		}

		for _, def := range page.Items {
			schemas = append(schemas, def.URI.String())
		}
		total = page.Total

		// NextOffset is zero on the last page. The second condition
		// guards against a driver that fails to advance the offset,
		// which would otherwise loop forever.
		if page.NextOffset == 0 || page.NextOffset <= offset {
			break
		}
		offset = page.NextOffset
	}

	sort.Strings(schemas)

	return &GetNamespaceResponse{
		Data:         ns,
		Schemas:      schemas,
		TotalSchemas: total,
	}, nil
}

// ListNamespacesRequest is the request for namespaces.list.
type ListNamespacesRequest struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ListNamespacesResponse is the response for namespaces.list.
type ListNamespacesResponse struct {
	Items      []string `json:"items"`
	NextOffset int      `json:"next_offset,omitempty"`
	Total      int      `json:"total"`
}

// List lists all known namespaces.
func (s *NamespaceService) List(ctx context.Context, req *ListNamespacesRequest) (*ListNamespacesResponse, error) {
	q := &store.Query{
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
