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
	ids []string,
) ([]*core.Record, []*core.URI, error) {
	columns := buildSQLColumns(d.schemaDef)

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

	schemaURI := d.schemaDef.URI()

	recordMap, err := d.scanSQLRows(rows, columns)
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
	columns []string,
) (map[string]*core.Record, error) {
	recordMap := make(map[string]*core.Record)

	for rows.Next() {
		record, id, err := d.scanSQLRow(rows, columns)
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
	columns []string,
) (*core.Record, string, error) {
	schemaURI := d.schemaDef.URI()
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

	for i, field := range d.schemaDef.Fields {
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

// TupleRef represents a reference to a specific tuple (id + attr) in a SQL table.
type TupleRef struct {
	ID   string
	Attr string
}

// GetTuples retrieves individual tuples from the SQL table.
func (d *SQLDriverTx) GetTuples(
	ctx context.Context,
	refs []TupleRef,
) ([]*core.Tuple, []*core.URI, error) {
	ids := uniqueIDs(refs)
	columns := buildSQLColumns(d.schemaDef)

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

	recordMap, err := d.scanSQLRows(rows, columns)
	if err != nil {
		return nil, nil, err
	}

	return extractTuplesFromRecords(d.schemaDef.URI(), recordMap, refs)
}

func uniqueIDs(refs []TupleRef) []string {
	idSet := make(map[string]bool)
	for _, ref := range refs {
		idSet[ref.ID] = true
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	return ids
}

func extractTuplesFromRecords(
	schemaURI *core.URI,
	recordMap map[string]*core.Record,
	refs []TupleRef,
) ([]*core.Tuple, []*core.URI, error) {
	tuples := make([]*core.Tuple, 0, len(refs))
	var missed []*core.URI

	for _, ref := range refs {
		record, ok := recordMap[ref.ID]
		if !ok {
			missed = append(missed, tupleRefURI(schemaURI, ref))
			continue
		}

		tuple := record.Get(ref.Attr)
		if tuple == nil {
			missed = append(missed, tupleRefURI(schemaURI, ref))
			continue
		}

		path := schemaURI.NS().String() + "/" + schemaURI.Schema().String() + "/" + ref.ID
		tuples = append(tuples, core.NewTuple(path, ref.Attr, tuple.Value()))
	}

	return tuples, missed, nil
}

func tupleRefURI(schemaURI *core.URI, ref TupleRef) *core.URI {
	return core.New().
		NS(schemaURI.NS().String()).
		Schema(schemaURI.Schema().String()).
		ID(ref.ID).
		Attr(ref.Attr).
		MustURI()
}

// PutTuples saves individual tuples to the SQL table.
func (d *SQLDriverTx) PutTuples(
	ctx context.Context,
	tuples []*core.Tuple,
) error {
	tbl := d.tableName()

	grouped := make(map[string][]*core.Tuple)
	for _, tuple := range tuples {
		id := tuple.ID().String()
		grouped[id] = append(grouped[id], tuple)
	}

	for id, idTuples := range grouped {
		columns := []string{"_id"}
		values := []any{id}

		for _, tuple := range idTuples {
			field := d.schemaDef.GetField(tuple.Attr().String())
			if field == nil {
				continue
			}

			sqlVal, err := valueToSQL(tuple.Value())
			if err != nil {
				return err
			}

			columns = append(columns, normalize(field.Name))
			values = append(values, sqlVal)
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

// DeleteTuples sets specific tuple columns to NULL in the SQL table.
func (d *SQLDriverTx) DeleteTuples(ctx context.Context, refs []TupleRef) error {
	tbl := d.tableName()

	grouped := make(map[string][]string)
	for _, ref := range refs {
		field := d.schemaDef.GetField(ref.Attr)
		if field == nil {
			continue
		}
		grouped[ref.ID] = append(grouped[ref.ID], normalize(field.Name))
	}

	for id, columns := range grouped {
		if err := d.queries.NullifyRecordColumns(ctx, internal.NullifyRecordColumnsParams{
			Table:   tbl,
			ID:      id,
			Columns: columns,
		}); err != nil {
			return err
		}
	}

	return nil
}

// AddDynamicFieldsFromTuples infers and adds new fields from tuples to a dynamic schema.
func (d *SQLDriverTx) AddDynamicFieldsFromTuples(
	ctx context.Context,
	tuples []*core.Tuple,
) (*schema.Def, error) {
	newFields, err := schema.InferFields(d.schemaDef, tuples)
	if err != nil {
		return nil, err
	}
	if len(newFields) > 0 {
		d.schemaDef.AddFields(newFields...)
		schemaURI := tuples[0].SchemaURI()
		return d.schemaDef, d.schemaDriver.PutSchema(ctx, schemaURI, d.schemaDef)
	}
	return d.schemaDef, nil
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
