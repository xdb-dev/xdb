package sql

import (
	"context"
	"fmt"
	"strings"
)

// KVRecord groups an ID with its attribute values.
type KVRecord struct {
	ID     string
	Values []Value
}

// CreateKVRecordParams are the arguments for [Queries.CreateKVRecord].
type CreateKVRecordParams struct {
	Table  string
	ID     string
	Values []Value
}

// CreateKVRecord inserts all values for a record into a KV table.
// Uses replace semantics: any existing rows for the ID are deleted first.
// Each value is stored natively via [Value] as a [driver.Valuer], with
// its [core.TID] (and array element id) recorded in _type/_elem.
func (q *Queries) CreateKVRecord(ctx context.Context, arg CreateKVRecordParams) error {
	delQuery := fmt.Sprintf("DELETE FROM %s WHERE _id = ?", arg.Table)
	if _, err := q.db.ExecContext(ctx, delQuery, arg.ID); err != nil {
		return mapErr(err)
	}

	if len(arg.Values) == 0 {
		return nil
	}

	rowPlaceholder := "(?, ?, ?, ?, ?)"
	placeholders := make([]string, len(arg.Values))
	args := make([]any, 0, len(arg.Values)*5)

	for i, v := range arg.Values {
		placeholders[i] = rowPlaceholder
		args = append(args, arg.ID, v.Name, string(v.Val.Type().ID()), v.ElemTID(), v)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (_id, _attr, _type, _elem, _val) VALUES %s",
		arg.Table,
		strings.Join(placeholders, ", "),
	)

	_, err := q.db.ExecContext(ctx, query, args...)
	return mapErr(err)
}

// GetKVRecordParams are the arguments for [Queries.GetKVRecord].
type GetKVRecordParams struct {
	Table string
	ID    string
}

// GetKVRecord retrieves all values for a record from a KV table.
// Returns nil, nil if no rows exist for the ID.
func (q *Queries) GetKVRecord(ctx context.Context, arg GetKVRecordParams) ([]Value, error) {
	query := fmt.Sprintf(
		"SELECT _attr, _type, _elem, _val FROM %s WHERE _id = ? ORDER BY _attr",
		arg.Table,
	)

	rows, err := q.db.QueryContext(ctx, query, arg.ID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close() //nolint:errcheck

	var result []Value
	for rows.Next() {
		v, err := scanKVValue(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}

// ListKVRecordsParams are the arguments for [Queries.ListKVRecords].
type ListKVRecordsParams struct {
	Table     string
	Where     string
	WhereArgs []any
	Offset    int
	Limit     int
}

// ListKVRecords lists records from a KV table, grouped by ID in order.
// The Limit/Offset apply to distinct record IDs via a subquery.
// When Where is set, it is injected into the inner ID-selecting subquery.
func (q *Queries) ListKVRecords(ctx context.Context, arg ListKVRecordsParams) ([]KVRecord, error) {
	var innerWhere string
	var queryArgs []any
	if arg.Where != "" {
		innerWhere = " AND " + arg.Where
		queryArgs = append(queryArgs, arg.WhereArgs...)
	}
	queryArgs = append(queryArgs, arg.Limit, arg.Offset)

	query := fmt.Sprintf(
		"SELECT _id, _attr, _type, _elem, _val FROM %s "+
			"WHERE _id IN (SELECT DISTINCT _id FROM %s WHERE 1=1%s ORDER BY _id LIMIT ? OFFSET ?) "+
			"ORDER BY _id, _attr",
		arg.Table, arg.Table, innerWhere,
	)

	rows, err := q.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close() //nolint:errcheck

	var result []KVRecord
	var cur *KVRecord

	for rows.Next() {
		var id string
		v, err := scanKVValueWithID(rows, &id)
		if err != nil {
			return nil, err
		}

		if cur == nil || cur.ID != id {
			result = append(result, KVRecord{ID: id})
			cur = &result[len(result)-1]
		}
		cur.Values = append(cur.Values, v)
	}

	return result, rows.Err()
}

// scanRow is the subset of [*sql.Rows] the KV scan helpers need.
type scanRow interface {
	Scan(dest ...any) error
}

// scanKVValue decodes one (_attr, _type, _elem, _val) row into a Value.
func scanKVValue(row scanRow) (Value, error) {
	var attr, tid, elem string
	var raw any
	if err := row.Scan(&attr, &tid, &elem, &raw); err != nil {
		return Value{}, err
	}
	return decodeKVValue(attr, tid, elem, raw)
}

// scanKVValueWithID decodes one (_id, _attr, _type, _elem, _val) row,
// writing the id through idOut.
func scanKVValueWithID(row scanRow, idOut *string) (Value, error) {
	var attr, tid, elem string
	var raw any
	if err := row.Scan(idOut, &attr, &tid, &elem, &raw); err != nil {
		return Value{}, err
	}
	return decodeKVValue(attr, tid, elem, raw)
}

// decodeKVValue reconstructs a typed Value from a KV row's columns.
func decodeKVValue(attr, tid, elem string, raw any) (Value, error) {
	v := Value{Name: attr, Type: TypeFrom(tid, elem)}
	if err := v.Scan(raw); err != nil {
		return Value{}, err
	}
	return v, nil
}

// DeleteKVRecordParams are the arguments for [Queries.DeleteKVRecord].
type DeleteKVRecordParams struct {
	Table string
	ID    string
}

// DeleteKVRecord deletes all rows for a record from a KV table.
func (q *Queries) DeleteKVRecord(ctx context.Context, arg DeleteKVRecordParams) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE _id = ?", arg.Table)
	_, err := q.db.ExecContext(ctx, query, arg.ID)
	return mapErr(err)
}

// KVRecordExistsParams are the arguments for [Queries.KVRecordExists].
type KVRecordExistsParams struct {
	Table string
	ID    string
}

// KVRecordExists checks if any rows exist for the given ID in a KV table.
func (q *Queries) KVRecordExists(ctx context.Context, arg KVRecordExistsParams) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE _id = ?)", arg.Table)
	var exists bool
	err := q.db.QueryRowContext(ctx, query, arg.ID).Scan(&exists)
	return exists, mapErr(err)
}

// CountKVRecordsParams are the arguments for [Queries.CountKVRecords].
type CountKVRecordsParams struct {
	Table     string
	Where     string // optional WHERE clause
	WhereArgs []any  // params for Where placeholders
}

// CountKVRecords returns the number of distinct records in a KV table.
// When Where is set, only IDs matching the filter are counted.
func (q *Queries) CountKVRecords(ctx context.Context, arg CountKVRecordsParams) (int, error) {
	var whereClause string
	if arg.Where != "" {
		whereClause = " WHERE " + arg.Where
	}

	query := fmt.Sprintf("SELECT COUNT(DISTINCT _id) FROM %s%s", arg.Table, whereClause)
	var count int
	err := q.db.QueryRowContext(ctx, query, arg.WhereArgs...).Scan(&count)
	return count, mapErr(err)
}
