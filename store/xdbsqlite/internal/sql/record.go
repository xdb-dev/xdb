package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/xdb-dev/xdb/core"
)

// UpsertRecordParams are the arguments for [Queries.UpsertRecord].
type UpsertRecordParams struct {
	Table  string
	ID     string
	Values []Value
}

// UpsertRecord inserts or updates a record in a column table.
func (q *Queries) UpsertRecord(ctx context.Context, arg UpsertRecordParams) error {
	cols := make([]string, 0, len(arg.Values)+1)
	placeholders := make([]string, 0, len(arg.Values)+1)
	updates := make([]string, 0, len(arg.Values))
	args := make([]any, 0, len(arg.Values)+1)

	cols = append(cols, "_id")
	placeholders = append(placeholders, "?")
	args = append(args, arg.ID)

	for _, v := range arg.Values {
		cols = append(cols, v.Name)
		placeholders = append(placeholders, "?")
		updates = append(updates, v.Name+" = excluded."+v.Name)
		args = append(args, v)
	}

	// A record with only _id has no columns to update on conflict;
	// "DO UPDATE SET" with an empty list is malformed SQL, so keep the
	// existing row with DO NOTHING.
	conflict := "DO NOTHING"
	if len(updates) > 0 {
		conflict = "DO UPDATE SET " + strings.Join(updates, ", ")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(_id) %s",
		arg.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		conflict,
	)

	_, err := q.db.ExecContext(ctx, query, args...)
	return mapErr(err)
}

// GetRecordParams are the arguments for [Queries.GetRecord].
type GetRecordParams struct {
	Table   string
	ID      string
	Columns []Value
}

// GetRecord retrieves a single record from a column table.
// Returns nil, nil if the record does not exist.
func (q *Queries) GetRecord(ctx context.Context, arg GetRecordParams) ([]Value, error) {
	vals := make([]Value, len(arg.Columns))
	colNames := make([]string, len(arg.Columns))
	ptrs := make([]any, len(arg.Columns))

	for i, c := range arg.Columns {
		vals[i] = Value{Name: c.Name, Type: c.Type}
		colNames[i] = c.Name
		ptrs[i] = &vals[i]
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE _id = ?",
		strings.Join(colNames, ", "),
		arg.Table,
	)

	if err := q.db.QueryRowContext(ctx, query, arg.ID).Scan(ptrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, mapErr(err)
	}

	return vals, nil
}

// ListRecordsParams are the arguments for [Queries.ListRecords].
type ListRecordsParams struct {
	Table     string
	Where     string
	Columns   []Value
	WhereArgs []any
	Offset    int
	Limit     int
}

// ListRecords lists records from a column table.
// Each row includes _id as the first Value (typed as string).
func (q *Queries) ListRecords(ctx context.Context, arg ListRecordsParams) ([][]Value, error) {
	colNames := make([]string, 0, len(arg.Columns)+1)
	colNames = append(colNames, "_id")
	for _, c := range arg.Columns {
		colNames = append(colNames, c.Name)
	}

	var whereClause string
	var queryArgs []any
	if arg.Where != "" {
		whereClause = " WHERE " + arg.Where
		queryArgs = append(queryArgs, arg.WhereArgs...)
	}
	queryArgs = append(queryArgs, arg.Limit, arg.Offset)

	query := fmt.Sprintf(
		"SELECT %s FROM %s%s ORDER BY _id LIMIT ? OFFSET ?",
		strings.Join(colNames, ", "),
		arg.Table,
		whereClause,
	)

	rows, err := q.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close() //nolint:errcheck

	// Reusable scanners: [0] = _id (raw string), [1..] = typed columns.
	nCols := len(colNames)
	scanners := make([]Value, len(arg.Columns))
	ptrs := make([]any, nCols)

	var idDest string
	ptrs[0] = &idDest
	for i, c := range arg.Columns {
		scanners[i] = Value{Name: c.Name, Type: c.Type}
		ptrs[i+1] = &scanners[i]
	}

	var result [][]Value
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}

		rowVals := make([]Value, nCols)
		rowVals[0] = Value{Name: "_id", Val: core.StringVal(idDest)}

		for i := range arg.Columns {
			rowVals[i+1] = scanners[i]
			scanners[i].Val = nil // reset for next row
		}

		result = append(result, rowVals)
	}

	return result, rows.Err()
}

// DeleteRecordParams are the arguments for [Queries.DeleteRecord].
type DeleteRecordParams struct {
	Table string
	ID    string
}

// DeleteRecord deletes a record from a column table.
func (q *Queries) DeleteRecord(ctx context.Context, arg DeleteRecordParams) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE _id = ?", arg.Table)
	_, err := q.db.ExecContext(ctx, query, arg.ID)
	return mapErr(err)
}

// RecordExistsParams are the arguments for [Queries.RecordExists].
type RecordExistsParams struct {
	Table string
	ID    string
}

// RecordExists checks if a record exists in a column table.
func (q *Queries) RecordExists(ctx context.Context, arg RecordExistsParams) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE _id = ?)", arg.Table)
	var exists bool
	err := q.db.QueryRowContext(ctx, query, arg.ID).Scan(&exists)
	return exists, mapErr(err)
}

// CountRecordsParams are the arguments for [Queries.CountRecords].
type CountRecordsParams struct {
	Table     string
	Where     string // optional WHERE clause (without "WHERE" keyword)
	WhereArgs []any  // params for Where placeholders
}

// CountRecords returns the number of records in a column table.
func (q *Queries) CountRecords(ctx context.Context, arg CountRecordsParams) (int, error) {
	var whereClause string
	if arg.Where != "" {
		whereClause = " WHERE " + arg.Where
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", arg.Table, whereClause)
	var count int
	err := q.db.QueryRowContext(ctx, query, arg.WhereArgs...).Scan(&count)
	return count, mapErr(err)
}
