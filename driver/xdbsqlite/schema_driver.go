package xdbsqlite

import (
	"context"
	"database/sql"
	"regexp"
	"time"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/driver/xdbsqlite/internal"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/x"
)

type SchemaDriverTx struct {
	tx      *sql.Tx
	queries *internal.Queries
}

func NewSchemaDriverTx(tx *sql.Tx) *SchemaDriverTx {
	return &SchemaDriverTx{
		tx:      tx,
		queries: internal.NewQueries(tx),
	}
}

func (d *SchemaDriverTx) GetSchema(ctx context.Context, uri *core.URI) (*schema.Def, error) {
	metadata, err := d.queries.GetMetadata(ctx, uri.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, driver.ErrNotFound
		}
		return nil, err
	}

	jsonSchema, err := schema.LoadFromJSON([]byte(metadata.Schema))
	if err != nil {
		return nil, err
	}

	return jsonSchema, nil
}

func (d *SchemaDriverTx) ListSchemas(ctx context.Context, uri *core.URI) ([]*schema.Def, error) {
	metadataList, err := d.queries.ListMetadata(ctx, uri.String())
	if err != nil {
		return nil, err
	}

	schemas := make([]*schema.Def, 0, len(metadataList))
	for _, metadata := range metadataList {
		def, err := schema.LoadFromJSON([]byte(metadata.Schema))
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, def)
	}

	return schemas, nil
}

func (d *SchemaDriverTx) PutSchema(ctx context.Context, uri *core.URI, def *schema.Def) error {
	tableName := tableName(uri)

	existing, err := d.GetSchema(ctx, uri)
	if err != nil && !errors.Is(err, driver.ErrNotFound) {
		return err
	}

	if existing != nil {
		columnsAdded, columnsRemoved, err := diffFields(existing, def)
		if err != nil {
			return err
		}

		if err = d.queries.AlterSQLTable(ctx, internal.AlterSQLTableParams{
			Name:        tableName,
			AddColumns:  columnsAdded,
			DropColumns: columnsRemoved,
		}); err != nil {
			return err
		}
	} else {
		columns, err := toSQLiteColumns(def.Fields)
		if err != nil {
			return err
		}

		err = d.queries.CreateSQLTable(ctx, internal.CreateSQLTableParams{
			Name:    tableName,
			Columns: columns,
		})
		if err != nil {
			return err
		}
	}

	jsonSchema, err := schema.WriteToJSON(def)
	if err != nil {
		return err
	}

	return d.queries.PutMetadata(ctx, internal.PutMetadataParams{
		URI:       uri.String(),
		Schema:    string(jsonSchema),
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	})
}

func (d *SchemaDriverTx) DeleteSchema(ctx context.Context, uri *core.URI) error {
	if err := d.queries.DeleteMetadata(ctx, uri.String()); err != nil {
		return err
	}

	return d.queries.DropTable(ctx, tableName(uri))
}

func tableName(uri *core.URI) string {
	return normalize(uri.NS().String() + "__" + uri.Schema().String())
}

var normalizeRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func normalize(name string) string {
	return normalizeRegex.ReplaceAllString(name, "_")
}

func sqliteTypeForField(field *schema.FieldDef) (string, error) {
	switch field.Type.ID() {
	case core.TIDString:
		return "TEXT", nil
	case core.TIDInteger:
		return "INTEGER", nil
	case core.TIDUnsigned:
		return "INTEGER", nil
	case core.TIDBoolean:
		return "INTEGER", nil
	case core.TIDFloat:
		return "REAL", nil
	case core.TIDBytes:
		return "BLOB", nil
	case core.TIDTime:
		return "INTEGER", nil
	case core.TIDArray:
		return "TEXT", nil
	case core.TIDMap:
		return "TEXT", nil
	default:
		return "", errors.Wrap(internal.ErrUnsupportedType, "type", field.Type.ID().String())
	}
}

func toSQLiteColumns(fields []*schema.FieldDef) ([][]string, error) {
	columns := make([][]string, len(fields))
	for i, field := range fields {
		sqlType, err := sqliteTypeForField(field)
		if err != nil {
			return nil, err
		}
		columns[i] = []string{normalize(field.Name), sqlType}
	}
	return columns, nil
}

func diffFields(existing, schemaDef *schema.Def) ([][]string, []string, error) {
	fieldsAdded := x.Diff(schemaDef.Fields, existing.Fields, byName)
	fieldsRemoved := x.Diff(existing.Fields, schemaDef.Fields, byName)

	columnsAdded, err := toSQLiteColumns(fieldsAdded)
	columnsRemoved := x.Map(fieldsRemoved, func(f *schema.FieldDef) string {
		return normalize(f.Name)
	})

	return columnsAdded, columnsRemoved, err
}

func byName(f *schema.FieldDef) string {
	return f.Name
}
