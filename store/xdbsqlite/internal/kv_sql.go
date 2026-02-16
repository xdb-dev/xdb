package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

// KVTuple represents a single tuple in the KV table.
type KVTuple struct {
	Key       string
	ID        string
	Attr      string
	Value     string
	UpdatedAt int64
}

// InsertKVTuplesParams contains parameters for inserting tuples into a KV table.
type InsertKVTuplesParams struct {
	Table  string
	Tuples []KVTuple
}

// InsertKVTuples inserts or updates multiple tuples in a KV table.
// It uses INSERT ... ON CONFLICT DO UPDATE to perform upserts.
func (q *Queries) InsertKVTuples(ctx context.Context, args InsertKVTuplesParams) error {
	if len(args.Tuples) == 0 {
		return nil
	}

	var query strings.Builder

	fmt.Fprintf(&query, `INSERT INTO "%s" (key, id, attr, value, updated_at) VALUES `, args.Table)

	values := make([]any, 0, len(args.Tuples)*5)
	for i, tuple := range args.Tuples {
		if i > 0 {
			query.WriteString(", ")
		}
		base := i * 5
		fmt.Fprintf(&query, "($%d, $%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5)
		values = append(values, tuple.Key, tuple.ID, tuple.Attr, tuple.Value, tuple.UpdatedAt)
	}

	query.WriteString(" ON CONFLICT (key) DO UPDATE SET ")
	query.WriteString("id = excluded.id, ")
	query.WriteString("attr = excluded.attr, ")
	query.WriteString("value = excluded.value, ")
	query.WriteString("updated_at = excluded.updated_at")

	_, err := q.tx.ExecContext(ctx, query.String(), values...)
	return err
}

// SelectKVTuplesParams contains parameters for selecting tuples from a KV table.
type SelectKVTuplesParams struct {
	Table string
	IDs   []string
}

// SelectKVTuples retrieves tuples by record IDs from a KV table.
// The caller is responsible for closing the returned rows.
func (q *Queries) SelectKVTuples(ctx context.Context, args SelectKVTuplesParams) (*sql.Rows, error) {
	if len(args.IDs) == 0 {
		return nil, errors.New("[xdbsqlite] no IDs provided")
	}

	var query strings.Builder

	fmt.Fprintf(&query, `SELECT key, id, attr, value, updated_at FROM "%s" WHERE id IN (`, args.Table)

	placeholders := make([]string, len(args.IDs))
	values := make([]any, len(args.IDs))
	for i, id := range args.IDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = id
	}
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(") ORDER BY id, attr")

	return q.tx.QueryContext(ctx, query.String(), values...)
}

// DeleteKVTuplesParams contains parameters for deleting tuples from a KV table.
type DeleteKVTuplesParams struct {
	Table string
	IDs   []string
}

// SelectKVTuplesByKeysParams contains parameters for selecting tuples by keys from a KV table.
type SelectKVTuplesByKeysParams struct {
	Table string
	Keys  []string
}

// SelectKVTuplesByKeys retrieves tuples by their composite keys from a KV table.
// The caller is responsible for closing the returned rows.
func (q *Queries) SelectKVTuplesByKeys(ctx context.Context, args SelectKVTuplesByKeysParams) (*sql.Rows, error) {
	if len(args.Keys) == 0 {
		return nil, errors.New("[xdbsqlite] no keys provided")
	}

	var query strings.Builder

	fmt.Fprintf(&query, `SELECT key, id, attr, value, updated_at FROM "%s" WHERE key IN (`, args.Table)

	placeholders := make([]string, len(args.Keys))
	values := make([]any, len(args.Keys))
	for i, key := range args.Keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = key
	}
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(") ORDER BY id, attr")

	return q.tx.QueryContext(ctx, query.String(), values...)
}

// DeleteKVTuplesByKeysParams contains parameters for deleting tuples by keys from a KV table.
type DeleteKVTuplesByKeysParams struct {
	Table string
	Keys  []string
}

// DeleteKVTuplesByKeys deletes tuples by their composite keys from a KV table.
func (q *Queries) DeleteKVTuplesByKeys(ctx context.Context, args DeleteKVTuplesByKeysParams) error {
	if len(args.Keys) == 0 {
		return nil
	}

	var query strings.Builder

	fmt.Fprintf(&query, `DELETE FROM "%s" WHERE key IN (`, args.Table)

	placeholders := make([]string, len(args.Keys))
	values := make([]any, len(args.Keys))
	for i, key := range args.Keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = key
	}
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(")")

	_, err := q.tx.ExecContext(ctx, query.String(), values...)
	return err
}

// DeleteKVTuples deletes tuples by record IDs from a KV table.
func (q *Queries) DeleteKVTuples(ctx context.Context, args DeleteKVTuplesParams) error {
	if len(args.IDs) == 0 {
		return nil
	}

	var query strings.Builder

	fmt.Fprintf(&query, `DELETE FROM "%s" WHERE id IN (`, args.Table)

	placeholders := make([]string, len(args.IDs))
	values := make([]any, len(args.IDs))
	for i, id := range args.IDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = id
	}
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(")")

	_, err := q.tx.ExecContext(ctx, query.String(), values...)
	return err
}
