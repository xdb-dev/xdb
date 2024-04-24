package sqlite

import (
	"context"
	"errors"
	"strings"

	"github.com/xdb-dev/xdb/schema"
	"golang.org/x/exp/maps"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitemigration"
	"zombiezen.com/go/sqlite/sqlitex"
)

const (
	allTablesQuery = `SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;`
)

type Migration struct {
	db     *sqlite.Conn
	schema *schema.Schema
}

func NewMigration(db *sqlite.Conn, schema *schema.Schema) *Migration {
	return &Migration{db: db, schema: schema}
}

// Generate generates the migration queries.
func (m *Migration) Generate(ctx context.Context) ([]string, error) {
	allTables, err := m.getAllTables(ctx)
	if err != nil {
		return nil, err
	}

	records := m.schema.Records
	recordsMap := make(map[string]schema.Record, len(records))

	for _, r := range records {
		recordsMap[r.Table] = r
	}

	recTableNames := extract(records, func(r schema.Record) string {
		return r.Table
	})

	dbTableNames := maps.Keys(allTables)

	tablesToDelete := diff(dbTableNames, recTableNames)
	if len(tablesToDelete) > 0 {
		return nil, errors.New("tables cannot be deleted - " + strings.Join(tablesToDelete, ", "))
	}

	tablesToCreate := diff(recTableNames, dbTableNames)
	commonTables := intersect(dbTableNames, recTableNames)
	queries := []string{}

	queries = append(queries, genCreateTableQueries(tablesToCreate, recordsMap)...)

	for _, table := range commonTables {
		dbCols := allTables[table]

		r := recordsMap[table]

		attrMap := make(map[string]schema.Attribute, len(r.Attributes))

		for _, a := range r.Attributes {
			attrMap[a.Name] = a
		}

		colNames := maps.Keys(dbCols)
		attrNames := extract(r.Attributes, func(a schema.Attribute) string {
			return a.Name
		})

		colsToDelete := diff(colNames, attrNames)
		if len(colsToDelete) > 0 {
			return nil, errors.New(table + " - columns cannot be deleted - " + strings.Join(colsToDelete, ", "))
		}

		// commonCols := intersect(colNames, attrNames)

		// for _, col := range commonCols {
		// 	attr := attrMap[col]
		// 	colType := dbCols[col]

		// 	if getDBType(attr) != colType.DatabaseTypeName() {
		// 		return nil, errors.New(table + " - " + col + " - type mismatch")
		// 	}
		// }

		colsToCreate := diff(attrNames, colNames)

		if len(colsToCreate) > 0 {
			queries = append(queries, genAlterTableQuery(table, colsToCreate, attrMap))
		}
	}

	return queries, nil
}

// Run executes the migration.
func (m *Migration) Run(ctx context.Context) error {
	queries, err := m.Generate(ctx)
	if err != nil {
		return err
	}

	sm := sqlitemigration.Schema{
		Migrations: queries,
	}

	err = sqlitemigration.Migrate(ctx, m.db, sm)
	if err != nil {
		return err
	}

	return nil
}

func (m *Migration) getAllTables(ctx context.Context) (map[string]map[string]string, error) {
	tables := make(map[string]map[string]string, 0)

	err := sqlitex.Execute(m.db, allTablesQuery, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			tables[stmt.ColumnText(0)] = make(map[string]string)

			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	for table := range tables {
		cols, err := m.getAllColumns(ctx, table)
		if err != nil {
			return nil, err
		}

		tables[table] = cols
	}

	return tables, nil
}

func (m *Migration) getAllColumns(_ context.Context, table string) (map[string]string, error) {
	colsMap := make(map[string]string, 0)
	query := `SELECT * FROM pragma_table_info('` + table + `');`

	err := sqlitex.Execute(m.db, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			name := stmt.ColumnText(1)
			typ := stmt.ColumnText(2)

			if name == "id" {
				return nil
			}

			colsMap[name] = typ

			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	return colsMap, nil
}

func genCreateTableQueries(tables []string, recordsMap map[string]schema.Record) []string {
	queries := make([]string, 0, len(tables))

	for _, table := range tables {
		r := recordsMap[table]
		queries = append(queries, genCreateTableQuery(r))
	}

	return queries
}

func genCreateTableQuery(r schema.Record) string {
	var sb strings.Builder

	sb.WriteString("CREATE TABLE IF NOT EXISTS ")
	sb.WriteString(r.Table)
	sb.WriteString(" (")
	sb.WriteString("id VARCHAR PRIMARY KEY, ")

	for i, a := range r.Attributes {
		sb.WriteString(a.Name)
		sb.WriteString(" ")
		sb.WriteString(getDBType(a))
		if i < len(r.Attributes)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(");")

	return sb.String()
}

func genAlterTableQuery(table string, cols []string, attrMap map[string]schema.Attribute) string {
	var sb strings.Builder

	sb.WriteString("ALTER TABLE ")
	sb.WriteString(table)
	sb.WriteString(" ")

	for i, col := range cols {
		attr := attrMap[col]
		sb.WriteString("ADD COLUMN ")
		sb.WriteString(col)
		sb.WriteString(" ")
		sb.WriteString(getDBType(attr))
		if i < len(cols)-1 {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(";")

	return sb.String()
}

func getDBType(a schema.Attribute) string {
	var dbType string

	switch a.Type {
	case schema.String:
		dbType = "VARCHAR"
	case schema.Int:
		dbType = "INTEGER"
	case schema.Float:
		dbType = "REAL"
	case schema.Bool:
		dbType = "BOOLEAN"
	case schema.Time:
		dbType = "INTEGER"
	}

	if a.Repeated {
		dbType += "[]"
	}

	return dbType
}

// diff returns a - b. It returns the elements that are in a but not in b.
func diff(a, b []string) []string {
	m := make(map[string]struct{}, len(b))
	for _, x := range b {
		m[x] = struct{}{}
	}

	var res []string
	for _, x := range a {
		if _, ok := m[x]; !ok {
			res = append(res, x)
		}
	}

	return res
}

func extract[T any, O any](records []O, f func(O) T) []T {
	res := make([]T, 0, len(records))
	for _, r := range records {
		res = append(res, f(r))
	}

	return res
}

func intersect(a, b []string) []string {
	m := make(map[string]struct{}, len(b))
	for _, x := range b {
		m[x] = struct{}{}
	}

	var res []string
	for _, x := range a {
		if _, ok := m[x]; ok {
			res = append(res, x)
		}
	}

	return res
}
