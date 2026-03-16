package sql

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Schema represents a row in the _schemas metadata table.
type Schema struct {
	Namespace string
	Schema    string
	Data      json.RawMessage
}

// PutSchemaParams are the arguments for [Queries.PutSchema].
type PutSchemaParams struct {
	Namespace string
	Schema    string
	Data      json.RawMessage
}

// PutSchema inserts or updates a schema definition.
func (q *Queries) PutSchema(ctx context.Context, arg PutSchemaParams) error {
	_, err := q.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO _schemas (_ns, _name, _data) VALUES (?, ?, ?)",
		arg.Namespace, arg.Schema, []byte(arg.Data),
	)
	return err
}

// GetSchemaParams are the arguments for [Queries.GetSchema].
type GetSchemaParams struct {
	Namespace string
	Schema    string
}

// GetSchema retrieves a schema definition by namespace and name.
// Returns nil, nil if no schema is found.
func (q *Queries) GetSchema(ctx context.Context, arg GetSchemaParams) (json.RawMessage, error) {
	var data []byte
	err := q.db.QueryRowContext(ctx,
		"SELECT _data FROM _schemas WHERE _ns = ? AND _name = ?",
		arg.Namespace, arg.Schema,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// ListSchemasParams are the arguments for [Queries.ListSchemas].
type ListSchemasParams struct {
	Namespace *string
	Offset    int
	Limit     int
}

// ListSchemas lists schemas, optionally scoped by namespace.
func (q *Queries) ListSchemas(ctx context.Context, arg ListSchemasParams) ([]Schema, error) {
	var query string
	var args []any

	if arg.Namespace != nil {
		query = "SELECT _ns, _name, _data FROM _schemas WHERE _ns = ? ORDER BY _ns, _name LIMIT ? OFFSET ?"
		args = []any{*arg.Namespace, arg.Limit, arg.Offset}
	} else {
		query = "SELECT _ns, _name, _data FROM _schemas ORDER BY _ns, _name LIMIT ? OFFSET ?"
		args = []any{arg.Limit, arg.Offset}
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []Schema
	for rows.Next() {
		var s Schema
		var data []byte
		if err := rows.Scan(&s.Namespace, &s.Schema, &data); err != nil {
			return nil, err
		}
		s.Data = json.RawMessage(data)
		result = append(result, s)
	}

	return result, rows.Err()
}

// DeleteSchemaParams are the arguments for [Queries.DeleteSchema].
type DeleteSchemaParams struct {
	Namespace string
	Schema    string
}

// DeleteSchema deletes a schema definition.
func (q *Queries) DeleteSchema(ctx context.Context, arg DeleteSchemaParams) error {
	_, err := q.db.ExecContext(ctx,
		"DELETE FROM _schemas WHERE _ns = ? AND _name = ?",
		arg.Namespace, arg.Schema,
	)
	return err
}

// SchemaExistsParams are the arguments for [Queries.SchemaExists].
type SchemaExistsParams struct {
	Namespace string
	Schema    string
}

// SchemaExists checks if a schema exists.
func (q *Queries) SchemaExists(ctx context.Context, arg SchemaExistsParams) (bool, error) {
	var exists bool
	err := q.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM _schemas WHERE _ns = ? AND _name = ?)",
		arg.Namespace, arg.Schema,
	).Scan(&exists)
	return exists, err
}
