package xdbsqlite

import (
	"context"
	"database/sql"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store/xdbsqlite/internal"
)

// SQLDriverTx implements SQL-based record operations for strict/dynamic mode schemas.
type SQLDriverTx struct {
	queries      *internal.Queries
	schemaDriver *SchemaDriverTx
	schemaDef    *schema.Def
}

// NewSQLDriverTx creates a new SQLDriverTx with the given transaction and schema definition.
func NewSQLDriverTx(tx *sql.Tx, schemaDef *schema.Def) *SQLDriverTx {
	return &SQLDriverTx{
		queries:      internal.NewQueries(tx),
		schemaDriver: NewSchemaStoreTx(tx),
		schemaDef:    schemaDef,
	}
}

func (d *SQLDriverTx) tableName() string {
	return normalize(d.schemaDef.NS.String() + "__" + d.schemaDef.Name)
}

// GetRecords retrieves records by IDs from the SQL table.
func (d *SQLDriverTx) GetRecords(
	ctx context.Context,
	schemaURI *core.URI,
	schemaDef *schema.Def,
	ids []string,
) ([]*core.Record, []*core.URI, error) {
	columns := buildSQLColumns(schemaDef)

	rows, err := d.queries.SelectRecords(ctx, internal.SelectRecordsParams{
		Table:   d.tableName(),
		Columns: columns,
		IDs:     ids,
	})
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	recordMap, err := d.scanSQLRows(rows, schemaURI, schemaDef, columns)
	if err != nil {
		return nil, nil, err
	}

	records := make([]*core.Record, 0, len(recordMap))
	var missed []*core.URI
	for _, id := range ids {
		if record, ok := recordMap[id]; ok {
			records = append(records, record)
		} else {
			missed = append(missed, core.New().
				NS(schemaURI.NS().String()).
				Schema(schemaURI.Schema().String()).
				ID(id).
				MustURI())
		}
	}

	return records, missed, nil
}

func buildSQLColumns(schemaDef *schema.Def) []string {
	columns := make([]string, 0, len(schemaDef.Fields)+1)
	columns = append(columns, "_id")
	for _, field := range schemaDef.Fields {
		columns = append(columns, normalize(field.Name))
	}
	return columns
}

func (d *SQLDriverTx) scanSQLRows(
	rows *sql.Rows,
	schemaURI *core.URI,
	schemaDef *schema.Def,
	columns []string,
) (map[string]*core.Record, error) {
	recordMap := make(map[string]*core.Record)

	for rows.Next() {
		record, id, err := d.scanSQLRow(rows, schemaURI, schemaDef, columns)
		if err != nil {
			return nil, err
		}
		recordMap[id] = record
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return recordMap, nil
}

func (d *SQLDriverTx) scanSQLRow(
	rows *sql.Rows,
	schemaURI *core.URI,
	schemaDef *schema.Def,
	columns []string,
) (*core.Record, string, error) {
	scanDest := make([]any, len(columns))
	var id string
	scanDest[0] = &id

	for i := 1; i < len(columns); i++ {
		var val any
		scanDest[i] = &val
	}

	if err := rows.Scan(scanDest...); err != nil {
		return nil, "", err
	}

	record := core.NewRecord(schemaURI.NS().String(), schemaURI.Schema().String(), id)

	for i, field := range schemaDef.Fields {
		sqlVal := *(scanDest[i+1].(*any))
		if sqlVal == nil {
			continue
		}

		coreVal, err := sqlToValue(sqlVal, field.Type)
		if err != nil {
			return nil, "", err
		}

		if coreVal != nil {
			record.Set(field.Name, coreVal)
		}
	}

	return record, id, nil
}

// PutRecords saves records to the SQL table.
func (d *SQLDriverTx) PutRecords(
	ctx context.Context,
	records []*core.Record,
) error {
	tbl := d.tableName()

	for _, record := range records {
		columns, values, err := buildRecordColumns(record, d.schemaDef)
		if err != nil {
			return err
		}

		if err := d.queries.InsertRecord(ctx, internal.InsertRecordParams{
			Table:   tbl,
			Columns: columns,
			Values:  values,
		}); err != nil {
			return err
		}
	}

	return nil
}

func buildRecordColumns(record *core.Record, schemaDef *schema.Def) ([]string, []any, error) {
	columns := make([]string, 0, len(schemaDef.Fields)+1)
	values := make([]any, 0, len(schemaDef.Fields)+1)

	columns = append(columns, "_id")
	values = append(values, record.ID().String())

	for _, field := range schemaDef.Fields {
		tuple := record.Get(field.Name)
		if tuple == nil {
			columns = append(columns, normalize(field.Name))
			values = append(values, nil)
			continue
		}

		sqlVal, err := valueToSQL(tuple.Value())
		if err != nil {
			return nil, nil, err
		}

		columns = append(columns, normalize(field.Name))
		values = append(values, sqlVal)
	}

	return columns, values, nil
}

// DeleteRecords removes records from the SQL table.
func (d *SQLDriverTx) DeleteRecords(ctx context.Context, ids []string) error {
	tbl := d.tableName()
	return d.queries.DeleteRecords(ctx, internal.DeleteRecordsParams{
		Table: tbl,
		IDs:   ids,
	})
}

// AddDynamicFields infers and adds new fields from records to a dynamic schema.
func (d *SQLDriverTx) AddDynamicFields(
	ctx context.Context,
	records []*core.Record,
) (*schema.Def, error) {
	var allTuples []*core.Tuple
	for _, record := range records {
		allTuples = append(allTuples, record.Tuples()...)
	}

	newFields, err := schema.InferFields(d.schemaDef, allTuples)
	if err != nil {
		return nil, err
	}
	if len(newFields) > 0 {
		d.schemaDef.AddFields(newFields...)
		schemaURI := records[0].SchemaURI()
		return d.schemaDef, d.schemaDriver.PutSchema(ctx, schemaURI, d.schemaDef)
	}
	return d.schemaDef, nil
}
