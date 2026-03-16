package sql

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdb-dev/xdb/core"
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
func (q *Queries) CreateKVRecord(ctx context.Context, arg CreateKVRecordParams) error {
	delQuery := fmt.Sprintf("DELETE FROM %s WHERE _id = ?", arg.Table)
	if _, err := q.db.ExecContext(ctx, delQuery, arg.ID); err != nil {
		return err
	}

	if len(arg.Values) == 0 {
		return nil
	}

	// Batch INSERT: one statement with multiple value rows.
	rowPlaceholder := "(?, ?, ?, ?)"
	placeholders := make([]string, len(arg.Values))
	args := make([]any, 0, len(arg.Values)*4)

	for i, v := range arg.Values {
		data, err := defaultCodec.ToBytes(v.Val)
		if err != nil {
			return err
		}
		placeholders[i] = rowPlaceholder
		args = append(args, arg.ID, v.Column, string(v.Val.Type().ID()), data)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (_id, _attr, _type, _val) VALUES %s",
		arg.Table,
		strings.Join(placeholders, ", "),
	)

	_, err := q.db.ExecContext(ctx, query, args...)
	return err
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
		"SELECT _attr, _type, _val FROM %s WHERE _id = ? ORDER BY _attr",
		arg.Table,
	)

	rows, err := q.db.QueryContext(ctx, query, arg.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []Value
	for rows.Next() {
		var attr, tid string
		var data []byte

		if err := rows.Scan(&attr, &tid, &data); err != nil {
			return nil, err
		}

		typ := core.NewType(core.TID(tid))
		cv, err := defaultCodec.FromBytes(typ, data)
		if err != nil {
			return nil, err
		}

		result = append(result, Value{Column: attr, Val: cv})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}

// ListKVRecordIDsParams are the arguments for [Queries.ListKVRecordIDs].
type ListKVRecordIDsParams struct {
	Table  string
	Offset int
	Limit  int
}

// ListKVRecordIDs lists distinct record IDs from a KV table.
func (q *Queries) ListKVRecordIDs(ctx context.Context, arg ListKVRecordIDsParams) ([]string, error) {
	query := fmt.Sprintf(
		"SELECT DISTINCT _id FROM %s ORDER BY _id LIMIT ? OFFSET ?",
		arg.Table,
	)

	rows, err := q.db.QueryContext(ctx, query, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}

	return result, rows.Err()
}

// ListKVRecordsParams are the arguments for [Queries.ListKVRecords].
type ListKVRecordsParams struct {
	Table  string
	Offset int
	Limit  int
}

// ListKVRecords lists records from a KV table, grouped by ID in order.
// The Limit/Offset apply to distinct record IDs via a subquery.
func (q *Queries) ListKVRecords(ctx context.Context, arg ListKVRecordsParams) ([]KVRecord, error) {
	query := fmt.Sprintf(
		"SELECT _id, _attr, _type, _val FROM %s "+
			"WHERE _id IN (SELECT DISTINCT _id FROM %s ORDER BY _id LIMIT ? OFFSET ?) "+
			"ORDER BY _id, _attr",
		arg.Table, arg.Table,
	)

	rows, err := q.db.QueryContext(ctx, query, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var result []KVRecord
	var cur *KVRecord

	for rows.Next() {
		var id, attr, tid string
		var data []byte

		if err := rows.Scan(&id, &attr, &tid, &data); err != nil {
			return nil, err
		}

		typ := core.NewType(core.TID(tid))
		cv, err := defaultCodec.FromBytes(typ, data)
		if err != nil {
			return nil, err
		}

		if cur == nil || cur.ID != id {
			result = append(result, KVRecord{ID: id})
			cur = &result[len(result)-1]
		}
		cur.Values = append(cur.Values, Value{Column: attr, Val: cv})
	}

	return result, rows.Err()
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
	return err
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
	return exists, err
}
