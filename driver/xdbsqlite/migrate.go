package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/gojekfarm/xtools/errors"

	"github.com/xdb-dev/xdb/types"
)

// Migrator is a migration manager for the SQLite driver.
type Migrator struct {
	db *sql.DB
}

// NewMigrator creates a new migration manager.
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// GenerateMigrations generates SQL migration statements for the given schema.
func (m *Migrator) GenerateMigrations(ctx context.Context, schemas []*types.Schema, w io.Writer) error {
	for _, schema := range schemas {
		tableName := schema.Kind
		exists, err := m.tableExists(ctx, tableName)
		if err != nil {
			return errors.Wrap(err, "kind", schema.Kind)
		}

		if exists {
			err = m.generateAlterTable(ctx, schema, w)
			if err != nil {
				return errors.Wrap(err, "kind", schema.Kind)
			}
		} else {
			err = m.generateCreateTable(schema, w)
			if err != nil {
				return errors.Wrap(err, "kind", schema.Kind)
			}
		}
	}

	return nil
}

func (m *Migrator) tableExists(ctx context.Context, name string) (bool, error) {
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"

	var count int

	err := m.db.QueryRowContext(ctx, query, name).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err)
	}

	return count > 0, nil
}

func (m *Migrator) generateCreateTable(schema *types.Schema, w io.Writer) error {
	tableName := schema.Kind

	up := []string{fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (`, tableName)}

	for i, attr := range schema.Attributes {
		sqlType, err := sqliteTypeForField(attr)
		if err != nil {
			return err
		}

		line := fmt.Sprintf(`	"%s" %s`, attr.Name, sqlType)
		if attr.PrimaryKey {
			line += " PRIMARY KEY"
		}

		if i < len(schema.Attributes)-1 {
			line += ","
		}

		up = append(up, line)
	}

	up = append(up, ");")

	_, err := w.Write([]byte(strings.Join(up, "\n")))
	if err != nil {
		return errors.Wrap(err, "kind", schema.Kind)
	}

	return nil
}

func (m *Migrator) generateAlterTable(ctx context.Context, schema *types.Schema, w io.Writer) error {
	tableName := schema.Kind
	up := []string{}

	existingCols, err := m.getTableColumns(ctx, tableName)
	if err != nil {
		return errors.Wrap(err, "kind", schema.Kind)
	}

	for _, attr := range schema.Attributes {
		if _, ok := existingCols[attr.Name]; !ok {
			sqlType, err := sqliteTypeForField(attr)
			if err != nil {
				return err
			}

			up = append(up, fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s;`, tableName, attr.Name, sqlType))
		}
	}

	_, err = w.Write([]byte(strings.Join(up, "\n")))
	if err != nil {
		return errors.Wrap(err, "kind", schema.Kind)
	}

	return nil
}

func (m *Migrator) getTableColumns(ctx context.Context, name string) (map[string]string, error) {
	query := "SELECT name, type FROM pragma_table_info(?)"

	rows, err := m.db.QueryContext(ctx, query, name)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rows.Close()

	existingCols := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, err
		}

		existingCols[name] = dataType
	}

	return existingCols, nil
}

// sqliteTypeForField maps types.Field to SQLite types.
func sqliteTypeForField(attr types.Attribute) (string, error) {
	switch attr.Type.ID() {
	case types.TypeIDString:
		return "TEXT", nil
	case types.TypeIDInteger,
		types.TypeIDBoolean,
		types.TypeIDTime:
		return "INTEGER", nil
	case types.TypeIDFloat:
		return "REAL", nil
	case types.TypeIDBytes:
		return "BLOB", nil
	case types.TypeIDArray:
		return "TEXT", nil
	case types.TypeIDMap:
		return "TEXT", nil
	default:
		return "", errors.Wrap(ErrUnsupportedValue, "type", attr.Type.Name())
	}
}
