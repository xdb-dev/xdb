package sql

import (
	"context"
)

// ListTablesParams are the arguments for [Queries.ListTables].
type ListTablesParams struct {
	// Pattern is a GLOB pattern matched against table names.
	// GLOB is used instead of LIKE so "_" in namespaces and schema
	// names matches literally.
	Pattern string
}

// ListTables lists user table names matching the pattern, sorted.
func (q *Queries) ListTables(ctx context.Context, arg ListTablesParams) ([]string, error) {
	rows, err := q.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name GLOB ? ORDER BY name",
		arg.Pattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}

	return result, rows.Err()
}
