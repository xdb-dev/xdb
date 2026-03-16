package sql

import "context"

// NamespaceExistsParams are the arguments for [Queries.NamespaceExists].
type NamespaceExistsParams struct {
	Namespace string
}

// NamespaceExists checks if any schema exists for the given namespace.
func (q *Queries) NamespaceExists(ctx context.Context, arg NamespaceExistsParams) (bool, error) {
	var exists bool
	err := q.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM _schemas WHERE _ns = ?)",
		arg.Namespace,
	).Scan(&exists)
	return exists, err
}

// ListNamespacesParams are the arguments for [Queries.ListNamespaces].
type ListNamespacesParams struct {
	Offset int
	Limit  int
}

// ListNamespaces lists distinct namespaces from the _schemas table.
func (q *Queries) ListNamespaces(ctx context.Context, arg ListNamespacesParams) ([]string, error) {
	rows, err := q.db.QueryContext(ctx,
		"SELECT DISTINCT _ns FROM _schemas ORDER BY _ns LIMIT ? OFFSET ?",
		arg.Limit, arg.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []string
	for rows.Next() {
		var ns string
		if err := rows.Scan(&ns); err != nil {
			return nil, err
		}
		result = append(result, ns)
	}

	return result, rows.Err()
}
