package xdbsqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gojekfarm/xtools/errors"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver"
	"github.com/xdb-dev/xdb/x"
)

var (
	ErrUnsupportedType = errors.New("xdb/driver/xdbsqlite: unsupported type")
	ErrFieldDeleted    = errors.New("xdb/driver/xdbsqlite: deleting fields is not supported")
	ErrFieldModified   = errors.New("xdb/driver/xdbsqlite: modifying field type or constraints is not supported")
)

func (s *Store) MakeRepo(ctx context.Context, repo *core.Repo) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	migrator := NewMigrator(tx)

	existing, err := s.repos.GetRepo(ctx, repo.Name())
	if err != nil && !errors.Is(err, driver.ErrRepoNotFound) {
		return err
	}

	var prev *core.Schema
	if existing != nil {
		prev = existing.Schema()
	}

	up, _, err := migrator.Generate(ctx, prev, repo.Schema())
	if err != nil {
		return err
	}

	_, err = tx.Exec(up)
	if err != nil {
		return err
	}

	err = s.repos.MakeRepo(ctx, repo)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetRepo(ctx context.Context, name string) (*core.Repo, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	migrator := NewMigrator(tx)

	exists, err := migrator.tableExists(ctx, name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, driver.ErrRepoNotFound
	}

	return s.repos.GetRepo(ctx, name)
}

func (s *Store) ListRepos(ctx context.Context) ([]*core.Repo, error) {
	return s.repos.ListRepos(ctx)
}

func (s *Store) DeleteRepo(ctx context.Context, name string) error {
	return s.repos.DeleteRepo(ctx, name)
}

type Migrator struct {
	tx *sql.Tx
}

func NewMigrator(tx *sql.Tx) *Migrator {
	return &Migrator{tx: tx}
}

func (m *Migrator) Generate(ctx context.Context, prev, next *core.Schema) (string, string, error) {
	tableName := next.Name

	exists, err := m.tableExists(ctx, tableName)
	if err != nil {
		return "", "", err
	}

	if prev == nil || !exists {
		return m.generateCreateTable(next)
	}

	return m.generateAlterTable(prev, next)
}

func (m *Migrator) tableExists(ctx context.Context, name string) (bool, error) {
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"

	var count int

	err := m.tx.QueryRowContext(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *Migrator) generateCreateTable(schema *core.Schema) (string, string, error) {
	tableName := schema.Name

	var up, down strings.Builder

	up.WriteString(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (`, tableName))

	count := len(schema.Fields)

	for i, field := range schema.Fields {
		sqlType, err := sqliteTypeForField(field)
		if err != nil {
			return "", "", err
		}

		up.WriteString(fmt.Sprintf(`	"%s" %s`, field.Name, sqlType))

		if field.Required {
			up.WriteString(" NOT NULL")
		}

		if i < count-1 {
			up.WriteString(",")
		}

		up.WriteString("\n")
	}

	up.WriteString(");")

	down.WriteString(fmt.Sprintf(`DROP TABLE IF EXISTS "%s";`, tableName))

	return up.String(), down.String(), nil
}

func (m *Migrator) generateAlterTable(prev, next *core.Schema) (string, string, error) {
	tableName := next.Name

	add, drop, modified := m.getDiffFields(prev, next)

	if len(drop) > 0 {
		names := x.Map(drop, func(field *core.Schema) string {
			return field.Name
		})
		return "", "", errors.Wrap(ErrFieldDeleted, "fields", strings.Join(names, ", "))
	}

	if len(modified) > 0 {
		names := x.Map(modified, func(field *core.Schema) string {
			return field.Name
		})
		return "", "", errors.Wrap(ErrFieldModified, "fields", strings.Join(names, ", "))
	}

	if len(add) == 0 {
		return "", "", nil
	}

	var up, down strings.Builder

	for _, field := range add {
		sqlType, err := sqliteTypeForField(field)
		if err != nil {
			return "", "", err
		}

		up.WriteString(fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s`, tableName, field.Name, sqlType))

		if field.Required {
			up.WriteString(" NOT NULL")
		}

		up.WriteString(";\n")

		down.WriteString(fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s";\n`, tableName, field.Name))
	}

	return up.String(), down.String(), nil
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

// fieldModified checks if a field's type or constraints have changed.
func fieldModified(prev, next *core.Schema) bool {
	return prev.Type != next.Type || prev.Required != next.Required
}

// sqliteTypeForField maps core.Field to SQLite core.
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
