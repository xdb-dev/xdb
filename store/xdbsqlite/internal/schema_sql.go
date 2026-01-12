package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
)

var ErrUnsupportedType = errors.New("[xdbsqlite] unsupported type")

type Queries struct {
	tx *sql.Tx
}

// NewQueries creates a new Queries instance with the given transaction.
func NewQueries(tx *sql.Tx) *Queries {
	return &Queries{tx: tx}
}

const createMetadataTableQuery = `
	CREATE TABLE IF NOT EXISTS "_xdb_metadata" (
		uri TEXT PRIMARY KEY,
		ns TEXT NOT NULL,
		schema TEXT,
		created_at INTEGER,
		updated_at INTEGER
	)
`

func (q *Queries) CreateMetadataTable(ctx context.Context) error {
	_, err := q.tx.ExecContext(ctx, createMetadataTableQuery)
	return err
}

const putMetadataQuery = `
	INSERT INTO "_xdb_metadata"
	(uri, ns, schema, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (uri) DO UPDATE SET
		ns = $2,
		schema = $3,
		updated_at = $5
`

type PutMetadataParams struct {
	URI       string
	NS        string
	Schema    string
	CreatedAt int64
	UpdatedAt int64
}

func (q *Queries) PutMetadata(ctx context.Context, args PutMetadataParams) error {
	_, err := q.tx.ExecContext(ctx, putMetadataQuery,
		args.URI, args.NS, args.Schema, args.CreatedAt, args.UpdatedAt,
	)
	return err
}

const getMetadataQuery = `
	SELECT uri, ns, schema, created_at, updated_at FROM "_xdb_metadata" WHERE uri = $1
`

func (q *Queries) GetMetadata(ctx context.Context, uri string) (*PutMetadataParams, error) {
	row := q.tx.QueryRowContext(ctx, getMetadataQuery, uri)
	var params PutMetadataParams
	err := row.Scan(&params.URI, &params.NS, &params.Schema, &params.CreatedAt, &params.UpdatedAt)
	return &params, err
}

const deleteMetadataQuery = `
	DELETE FROM "_xdb_metadata" WHERE uri = $1
`

func (q *Queries) DeleteMetadata(ctx context.Context, uri string) error {
	_, err := q.tx.ExecContext(ctx, deleteMetadataQuery, uri)
	return err
}

const listMetadataQuery = `
	SELECT uri, ns, schema, created_at, updated_at FROM "_xdb_metadata" WHERE uri LIKE $1
`

func (q *Queries) ListMetadata(ctx context.Context, uriPrefix string) ([]*PutMetadataParams, error) {
	rows, err := q.tx.QueryContext(ctx, listMetadataQuery, uriPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var results []*PutMetadataParams
	for rows.Next() {
		var params PutMetadataParams
		if err := rows.Scan(&params.URI, &params.NS, &params.Schema, &params.CreatedAt, &params.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, &params)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

const listNamespacesQuery = `
	SELECT DISTINCT ns FROM "_xdb_metadata" ORDER BY ns
`

func (q *Queries) ListNamespaces(ctx context.Context) ([]string, error) {
	rows, err := q.tx.QueryContext(ctx, listNamespacesQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var namespaces []string
	for rows.Next() {
		var ns string
		if err := rows.Scan(&ns); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return namespaces, nil
}

const createKVTableQuery = `
	CREATE TABLE IF NOT EXISTS "%s" (
		key TEXT PRIMARY KEY,
		id TEXT,
		attr TEXT,
		value TEXT,
		updated_at INTEGER
	)
`

func (q *Queries) CreateKVTable(ctx context.Context, name string) error {
	query := fmt.Sprintf(createKVTableQuery, name)
	_, err := q.tx.ExecContext(ctx, query)
	return err
}

type CreateSQLTableParams struct {
	Name    string
	Columns [][]string
}

func (q *Queries) CreateSQLTable(ctx context.Context, args CreateSQLTableParams) error {
	var query strings.Builder

	query.WriteString("CREATE TABLE IF NOT EXISTS ")
	query.WriteString(args.Name)
	query.WriteString(" (\n")

	count := len(args.Columns)

	for i, column := range args.Columns {
		query.WriteString("\t")
		query.WriteString(strings.Join(column, " "))

		if i < count-1 {
			query.WriteString(",")
		}

		query.WriteString("\n")
	}

	query.WriteString(");")

	_, err := q.tx.ExecContext(ctx, query.String())
	if err != nil {
		return err
	}

	return nil
}

type AlterSQLTableParams struct {
	Name        string
	AddColumns  [][]string
	DropColumns []string
}

func (q *Queries) AlterSQLTable(ctx context.Context, args AlterSQLTableParams) error {
	var query strings.Builder

	for _, column := range args.AddColumns {
		query.WriteString("ALTER TABLE ")
		query.WriteString(args.Name)
		query.WriteString(" ADD COLUMN ")
		query.WriteString(strings.Join(column, " "))
		query.WriteString(";\n")
	}

	for _, column := range args.DropColumns {
		query.WriteString("ALTER TABLE ")
		query.WriteString(args.Name)
		query.WriteString(" DROP COLUMN ")
		query.WriteString(column)
		query.WriteString(";\n")
	}

	_, err := q.tx.ExecContext(ctx, query.String())

	return err
}

func (q *Queries) DropTable(ctx context.Context, name string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s;", name)
	_, err := q.tx.ExecContext(ctx, query)
	return err
}
