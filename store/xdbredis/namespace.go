package xdbredis

import (
	"context"
	"fmt"
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// GetNamespace checks if a namespace exists.
func (s *Store) GetNamespace(ctx context.Context, uri *core.URI) (string, error) {
	ns := uri.NS()

	exists, err := s.client.SIsMember(ctx, s.nsIndexKey(), ns).Result()
	if err != nil {
		return "", fmt.Errorf("xdbredis: check namespace: %w", err)
	}
	if !exists {
		return "", store.ErrNotFound
	}

	return ns, nil
}

// ListNamespaces lists all known namespaces.
func (s *Store) ListNamespaces(
	ctx context.Context,
	q *store.Query,
) (*store.Page[string], error) {
	names, err := s.client.SMembers(ctx, s.nsIndexKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: list namespaces: %w", err)
	}

	sort.Strings(names)

	return store.Paginate(names, q), nil
}
