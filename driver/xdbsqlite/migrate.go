package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

var (
	// ErrUnsupportedType is returned when a XDB type is not supported by SQLite.
	ErrUnsupportedType = errors.New("[xdbsqlite] unsupported XDB type")
	// ErrFieldDeleted is returned when a field is deleted from a schema.
	ErrFieldDeleted = errors.New("[xdbsqlite] deleting fields is not supported")
	// ErrFieldModified is returned when a field is modified in a schema.
	ErrFieldModified = errors.New("[xdbsqlite] modifying field type or constraints is not supported")
)

// Migrator manages the schema generation and migrations
type Migrator struct {
	tx *sql.Tx
}

// CreateTable creates a new table for the given schema.
func (r *Migrator) CreateTable(ctx context.Context, tableName string, schema *core.Schema) error {
	query, err := r.generateCreateTable(tableName, schema)
	if err != nil {
		return err
	}

	_, err = r.tx.ExecContext(ctx, query)

	return err
}

// CreateKeyValueTable creates a new key-value table for the given name.
func (r *Migrator) CreateKeyValueTable(ctx context.Context, name string) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (
		key TEXT PRIMARY KEY,
		id TEXT,
		attr TEXT,
		value TEXT,
		updated_at INTEGER
	)`, name)
	_, err := r.tx.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	idxID := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_id ON %s (id);", name, name)
	_, err = r.tx.ExecContext(ctx, idxID)
	if err != nil {
		return err
	}

	idxAttr := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_attr ON %s (attr);", name, name)
	_, err = r.tx.ExecContext(ctx, idxAttr)
	if err != nil {
		return err
	}

	return nil
}

// AlterTable alters the table by adding fields.
// Deleting or modifying fields is not supported.
func (r *Migrator) AlterTable(ctx context.Context, tableName string, prev, next *core.Schema) error {
	query, err := r.generateAlterTable(tableName, prev, next)
	if err != nil {
		return err
	}

	_, err = r.tx.ExecContext(ctx, query)

	return err
}

// DropTable drops the table for the given name.
func (r *Migrator) DropTable(ctx context.Context, name string) error {
	query := fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`, name)
	_, err := r.tx.ExecContext(ctx, query)
	return err
}

func (r *Migrator) generateCreateTable(tableName string, schema *core.Schema) (string, error) {

	var query strings.Builder

	query.WriteString(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (`, tableName))

	count := len(schema.Fields)

	for i, field := range schema.Fields {
		sqlType, err := sqliteTypeForField(field)
		if err != nil {
			return "", err
		}

		query.WriteString(fmt.Sprintf(`	"%s" %s`, field.Name, sqlType))

		if slices.Contains(schema.Required, field.Name) {
			query.WriteString(" NOT NULL")
		}

		if field.Default != nil {
			query.WriteString(fmt.Sprintf(" DEFAULT \"%s\"", field.Default.String()))
		}

		if i < count-1 {
			query.WriteString(",")
		}

		query.WriteString("\n")
	}

	query.WriteString(");")

	return query.String(), nil
}

func (r *Migrator) generateAlterTable(tableName string, prev, next *core.Schema) (string, error) {

	add, drop, modified := r.getDiffFields(prev, next)

	if len(drop) > 0 {
		names := x.Map(drop, func(field *core.FieldSchema) string {
			return field.Name
		})
		return "", errors.Wrap(ErrFieldDeleted, "fields", strings.Join(names, ", "))
	}

	if len(modified) > 0 {
		names := x.Map(modified, func(field *core.FieldSchema) string {
			return field.Name
		})
		return "", errors.Wrap(ErrFieldModified, "fields", strings.Join(names, ", "))
	}

	if len(add) == 0 {
		return "", nil
	}

	var query strings.Builder

	for _, field := range add {
		sqlType, err := sqliteTypeForField(field)
		if err != nil {
			return "", err
		}

		query.WriteString(fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, tableName, field.Name, sqlType))

		if slices.Contains(next.Required, field.Name) {
			query.WriteString(" NOT NULL")
		}

		if field.Default != nil {
			query.WriteString(fmt.Sprintf(" DEFAULT \"%s\"", field.Default.String()))
		}

		query.WriteString(";\n")
	}

	return query.String(), nil
}

func (r *Migrator) getDiffFields(prev, next *core.Schema) ([]*core.FieldSchema, []*core.FieldSchema, []*core.FieldSchema) {
	add := []*core.FieldSchema{}
	drop := []*core.FieldSchema{}
	modified := []*core.FieldSchema{}

	nextFields := x.Index(next.Fields, func(field *core.FieldSchema) string {
		return field.Name
	})
	prevFields := x.Index(prev.Fields, func(field *core.FieldSchema) string {
		return field.Name
	})

	for _, field := range next.Fields {
		prevField, existsInPrev := prevFields[field.Name]
		if !existsInPrev {
			add = append(add, field)
		} else if !prevField.Equals(field) {
			modified = append(modified, field)
		}
	}

	for _, field := range prev.Fields {
		if _, ok := nextFields[field.Name]; !ok {
			drop = append(drop, field)
		}
	}

	return add, drop, modified
}

func sqliteTypeForField(field *core.FieldSchema) (string, error) {
	switch field.Type.ID() {
	case core.TypeIDString:
		return "TEXT", nil
	case core.TypeIDInteger:
		return "INTEGER", nil
	case core.TypeIDBoolean:
		return "INTEGER", nil
	case core.TypeIDTime:
		return "INTEGER", nil
	case core.TypeIDFloat:
		return "REAL", nil
	case core.TypeIDBytes:
		return "BLOB", nil
	case core.TypeIDArray:
		return "TEXT", nil
	case core.TypeIDMap:
		return "TEXT", nil
	default:
		return "", errors.Wrap(ErrUnsupportedType,
			"type", field.Type.String(),
			"field", field.Name,
		)
	}
}
