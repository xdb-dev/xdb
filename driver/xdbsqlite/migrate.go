package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/x"
)

// Migrator manages the schema generation and applying SQLite
// migrations to the database.
type Migrator struct {
	tx *sql.Tx
}

func NewMigrator(tx *sql.Tx) *Migrator {
	return &Migrator{tx: tx}
}

func (m *Migrator) CreateTable(ctx context.Context, schema *core.Schema) error {
	query, err := m.generateCreateTable(schema)
	if err != nil {
		return err
	}

	_, err = m.tx.ExecContext(ctx, query)

	return err
}

func (m *Migrator) AlterTable(ctx context.Context, prev, next *core.Schema) error {
	query, err := m.generateAlterTable(prev, next)
	if err != nil {
		return err
	}

	_, err = m.tx.ExecContext(ctx, query)

	return err
}

func (m *Migrator) DropTable(ctx context.Context, name string) error {
	query := fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`, name)
	_, err := m.tx.ExecContext(ctx, query)
	return err
}

func (m *Migrator) generateCreateTable(schema *core.Schema) (string, error) {
	tableName := schema.Name

	var query strings.Builder

	query.WriteString(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (`, tableName))

	count := len(schema.Fields)

	for i, field := range schema.Fields {
		sqlType, err := sqliteTypeForField(field)
		if err != nil {
			return "", err
		}

		query.WriteString(fmt.Sprintf(`	"%s" %s`, field.Name, sqlType))

		if field.Required {
			query.WriteString(" NOT NULL")
		}

		if i < count-1 {
			query.WriteString(",")
		}

		query.WriteString("\n")
	}

	query.WriteString(");")

	return query.String(), nil
}

func (m *Migrator) generateAlterTable(prev, next *core.Schema) (string, error) {
	tableName := next.Name

	add, drop, modified := m.getDiffFields(prev, next)

	if len(drop) > 0 {
		names := x.Map(drop, func(field *core.Schema) string {
			return field.Name
		})
		return "", errors.Wrap(ErrFieldDeleted, "fields", strings.Join(names, ", "))
	}

	if len(modified) > 0 {
		names := x.Map(modified, func(field *core.Schema) string {
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

		if field.Required {
			query.WriteString(" NOT NULL")
		}

		query.WriteString(";\n")
	}

	return query.String(), nil
}

func (m *Migrator) getDiffFields(prev, next *core.Schema) ([]*core.Schema, []*core.Schema, []*core.Schema) {
	add := []*core.Schema{}
	drop := []*core.Schema{}
	modified := []*core.Schema{}

	nextFields := x.Index(next.Fields, func(field *core.Schema) string {
		return field.Name
	})
	prevFields := x.Index(prev.Fields, func(field *core.Schema) string {
		return field.Name
	})

	for _, field := range next.Fields {
		prevField, existsInPrev := prevFields[field.Name]
		if !existsInPrev {
			add = append(add, field)
		} else if fieldModified(prevField, field) {
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

func fieldModified(prev, next *core.Schema) bool {
	return prev.Type != next.Type || prev.Required != next.Required
}

func sqliteTypeForField(field *core.Schema) (string, error) {
	switch field.Type {
	case core.TypeIDString.String():
		return "TEXT", nil
	case core.TypeIDInteger.String():
		return "INTEGER", nil
	case core.TypeIDBoolean.String():
		return "INTEGER", nil
	case core.TypeIDTime.String():
		return "INTEGER", nil
	case core.TypeIDFloat.String():
		return "REAL", nil
	case core.TypeIDBytes.String():
		return "BLOB", nil
	case core.TypeIDArray.String():
		return "TEXT", nil
	case core.TypeIDMap.String():
		return "TEXT", nil
	default:
		return "", errors.Wrap(ErrUnsupportedType, "type", field.Type)
	}
}
