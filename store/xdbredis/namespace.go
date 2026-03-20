package xdbredis

import (
	"context"
	"fmt"
	"sort"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
)

// GetNamespace checks if a namespace exists.
func (s *Store) GetNamespace(ctx context.Context, uri *core.URI) (*core.NS, error) {
	ns := uri.NS()

	exists, err := s.client.SIsMember(ctx, s.nsIndexKey(), ns.String()).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: check namespace: %w", err)
	}
	if !exists {
		return nil, store.ErrNotFound
	}

	return ns, nil
}

// ListNamespaces lists all known namespaces.
func (s *Store) ListNamespaces(
	ctx context.Context,
	q *store.Query,
) (*store.Page[*core.NS], error) {
	names, err := s.client.SMembers(ctx, s.nsIndexKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("xdbredis: list namespaces: %w", err)
	}

	sort.Strings(names)

	items := make([]*core.NS, len(names))
	for i, name := range names {
		ns, err := core.ParseNS(name)
		if err != nil {
			return nil, fmt.Errorf("xdbredis: parse namespace %s: %w", name, err)
		}
		items[i] = ns
	}

	return store.Paginate(items, q), nil
}
