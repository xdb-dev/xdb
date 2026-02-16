package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

// InsertRecordParams contains parameters for inserting a record into a SQL table.
type InsertRecordParams struct {
	Table   string
	Columns []string
	Values  []any
}

// InsertRecord inserts or updates a record in a SQL table.
// It uses INSERT ... ON CONFLICT DO UPDATE to perform an upsert.
// The first column is expected to be _id (the primary key).
func (q *Queries) InsertRecord(ctx context.Context, args InsertRecordParams) error {
	if len(args.Columns) == 0 {
		return errors.New("[xdbsqlite] no columns provided")
	}

	if len(args.Columns) != len(args.Values) {
		return errors.New("[xdbsqlite] columns and values count mismatch")
	}

	var query strings.Builder

	query.WriteString("INSERT INTO ")
	query.WriteString(args.Table)
	query.WriteString(" (")
	query.WriteString(strings.Join(args.Columns, ", "))
	query.WriteString(") VALUES (")

	placeholders := make([]string, len(args.Columns))
	for i := range args.Columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(")")

	if len(args.Columns) > 1 {
		query.WriteString(" ON CONFLICT (")
		query.WriteString(args.Columns[0])
		query.WriteString(") DO UPDATE SET ")

		updates := make([]string, 0, len(args.Columns)-1)
		for i := 1; i < len(args.Columns); i++ {
			updates = append(updates, fmt.Sprintf("%s = $%d", args.Columns[i], i+1))
		}
		query.WriteString(strings.Join(updates, ", "))
	} else {
		query.WriteString(" ON CONFLICT (")
		query.WriteString(args.Columns[0])
		query.WriteString(") DO NOTHING")
	}

	_, err := q.tx.ExecContext(ctx, query.String(), args.Values...)
	return err
}

// SelectRecordsParams contains parameters for selecting records from a SQL table.
type SelectRecordsParams struct {
	Table   string
	Columns []string
	IDs     []string
}

// SelectRecords retrieves records by IDs from a SQL table.
// The caller is responsible for closing the returned rows.
func (q *Queries) SelectRecords(ctx context.Context, args SelectRecordsParams) (*sql.Rows, error) {
	if len(args.IDs) == 0 {
		return nil, errors.New("[xdbsqlite] no IDs provided")
	}

	var query strings.Builder

	query.WriteString("SELECT ")
	if len(args.Columns) > 0 {
		query.WriteString(strings.Join(args.Columns, ", "))
	} else {
		query.WriteString("*")
	}
	query.WriteString(" FROM ")
	query.WriteString(args.Table)
	query.WriteString(" WHERE _id IN (")

	placeholders := make([]string, len(args.IDs))
	values := make([]any, len(args.IDs))
	for i, id := range args.IDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		values[i] = id
	}
	query.WriteString(strings.Join(placeholders, ", "))
	query.WriteString(")")

	return q.tx.QueryContext(ctx, query.String(), values...)
}

// DeleteRecordsParams contains parameters for deleting records from a SQL table.
type DeleteRecordsParams struct {
	Table string
	IDs   []string
}

// DeleteRecords deletes records by IDs from a SQL table.
func (q *Queries) DeleteRecords(ctx context.Context, args DeleteRecordsParams) error {
	if len(args.IDs) == 0 {
		return nil
	}

	var query strings.Builder

	query.WriteString("DELETE FROM ")
	query.WriteString(args.Table)
	query.WriteString(" WHERE _id IN (")

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
