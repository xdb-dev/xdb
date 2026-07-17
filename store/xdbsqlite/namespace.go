package xdbsqlite

import (
	"context"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store"
	xsql "github.com/xdb-dev/xdb/store/xdbsqlite/internal/sql"
)

// NamespaceTx holds a Queries instance for namespace operations.
type NamespaceTx struct {
	q *xsql.Queries
}

func (n *NamespaceTx) GetNamespace(ctx context.Context, uri *core.URI) (string, error) {
	exists, err := n.q.NamespaceExists(ctx, xsql.NamespaceExistsParams{
		Namespace: uri.NS(),
	})
	if err != nil {
		return "", err
	}
	if !exists {
		return "", store.ErrNotFound
	}

	return uri.NS(), nil
}

func (n *NamespaceTx) ListNamespaces(ctx context.Context, q *store.Query) (*store.Page[string], error) {
	names, err := n.q.ListNamespaces(ctx, xsql.ListNamespacesParams{
		Limit: 10000,
	})
	if err != nil {
		return nil, err
	}

	return store.Paginate(names, q), nil
}

// --- Store delegation ---

// GetNamespace checks if any schema exists in the given namespace.
func (s *Store) GetNamespace(ctx context.Context, uri *core.URI) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	ntx := &NamespaceTx{q: xsql.NewQueries(tx)}

	res, err := ntx.GetNamespace(ctx, uri)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return res, nil
}

// ListNamespaces lists unique namespaces derived from schemas.
func (s *Store) ListNamespaces(
	ctx context.Context,
	q *store.Query,
) (*store.Page[string], error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	ntx := &NamespaceTx{q: xsql.NewQueries(tx)}

	res, err := ntx.ListNamespaces(ctx, q)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return res, nil
}
