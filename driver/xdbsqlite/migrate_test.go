package xdbsqlite

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/core"
)

var (
	allTypesSchemaName = "com.example.AllTypes"
	allTypesSchema     = &core.Schema{
		Fields: []*core.FieldSchema{
		{
			Name:    "field_string",
			Type:    core.TypeString,
			Default: core.NewValue("default string"),
		},
		{
			Name:    "field_integer",
			Type:    core.TypeInt,
			Default: core.NewValue(int64(100)),
		},
		{
			Name:    "field_boolean",
			Type:    core.TypeBool,
			Default: core.NewValue(true),
		},
		{
			Name:    "field_time",
			Type:    core.TypeTime,
			Default: core.NewValue(time.Now()),
		},
		{
			Name:    "field_float",
			Type:    core.TypeFloat,
			Default: core.NewValue(3.14),
		},
		{
			Name:    "field_bytes",
			Type:    core.TypeBytes,
			Default: core.NewValue([]byte("hello")),
		},
		{
			Name:    "field_array",
			Type:    core.NewArrayType(core.TypeIDString),
			Default: core.NewValue([]string{"hello", "world"}),
		},
		{
			Name:    "field_map",
			Type:    core.NewMapType(core.TypeIDString, core.TypeIDString),
			Default: core.NewValue(map[string]string{"hello": "world"}),
		},
	},
	}
)

type MigratorTestSuite struct {
	suite.Suite
	db *sql.DB
}

func TestMigratorSuite(t *testing.T) {
	suite.Run(t, new(MigratorTestSuite))
}

func (s *MigratorTestSuite) SetupSuite() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)
	s.db = db
}

func (s *MigratorTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *MigratorTestSuite) TestCreateTable() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)

	migrator := &Migrator{tx: tx}

	err = migrator.CreateTable(s.T().Context(), allTypesSchemaName, allTypesSchema)
	s.Require().NoError(err)

	err = tx.Commit()
	s.Require().NoError(err)

	rows, err := s.db.Query(`SELECT * FROM "` + allTypesSchemaName + `"`)
	s.Require().NoError(err)
	defer rows.Close()

	gotColumns, err := rows.Columns()
	s.Require().NoError(err)

	gotTypes, err := rows.ColumnTypes()
	s.Require().NoError(err)

	expectedColumns := []struct {
		Name string
		Type string
	}{
		{Name: "field_string", Type: "TEXT"},
		{Name: "field_integer", Type: "INTEGER"},
		{Name: "field_boolean", Type: "INTEGER"},
		{Name: "field_time", Type: "INTEGER"},
		{Name: "field_float", Type: "REAL"},
		{Name: "field_bytes", Type: "BLOB"},
		{Name: "field_array", Type: "TEXT"},
		{Name: "field_map", Type: "TEXT"},
	}

	for i, column := range gotColumns {
		s.Equal(expectedColumns[i].Name, column)
		s.Equal(expectedColumns[i].Type, gotTypes[i].DatabaseTypeName())
	}
}

func (s *MigratorTestSuite) TestAlterTable_NoChange() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := &Migrator{tx: tx}

	err = migrator.CreateTable(s.T().Context(), allTypesSchemaName, allTypesSchema)
	s.Require().NoError(err)

	err = migrator.AlterTable(s.T().Context(), allTypesSchemaName, allTypesSchema, allTypesSchema)
	s.Require().NoError(err)

	rows, err := tx.Query(`SELECT * FROM "` + allTypesSchemaName + `"`)
	s.Require().NoError(err)
	defer rows.Close()

	gotColumns, err := rows.Columns()
	s.Require().NoError(err)

	expectedColumns := []string{
		"field_string",
		"field_integer",
		"field_boolean",
		"field_time",
		"field_float",
		"field_bytes",
		"field_array",
		"field_map",
	}

	s.Equal(expectedColumns, gotColumns)
}

func (s *MigratorTestSuite) TestAlterTable_AddFields() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := &Migrator{tx: tx}

	err = migrator.CreateTable(s.T().Context(), allTypesSchemaName, allTypesSchema)
	s.Require().NoError(err)

	extendedSchema := allTypesSchema.Clone()
	extendedSchema.Fields = append(extendedSchema.Fields, &core.FieldSchema{
		Name:    "author",
		Type:    core.TypeString,
		Default: core.NewValue("John Doe"),
	})

	err = migrator.AlterTable(s.T().Context(), allTypesSchemaName, allTypesSchema, extendedSchema)
	s.Require().NoError(err)

	err = tx.Commit()
	s.Require().NoError(err)

	rows, err := s.db.Query(`SELECT * FROM "` + allTypesSchemaName + `"`)
	s.Require().NoError(err)
	defer rows.Close()

	gotColumns, err := rows.Columns()
	s.Require().NoError(err)

	expectedColumns := []string{
		"field_string",
		"field_integer",
		"field_boolean",
		"field_time",
		"field_float",
		"field_bytes",
		"field_array",
		"field_map",
		"author",
	}

	s.Equal(expectedColumns, gotColumns)
}

func (s *MigratorTestSuite) TestAlterTable_DropFields() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := &Migrator{tx: tx}

	err = migrator.CreateTable(s.T().Context(), allTypesSchemaName, allTypesSchema)
	s.Require().NoError(err)

	reducedSchema := allTypesSchema.Clone()
	reducedSchema.Fields = reducedSchema.Fields[:len(reducedSchema.Fields)-1]

	err = migrator.AlterTable(s.T().Context(), allTypesSchemaName, allTypesSchema, reducedSchema)
	s.ErrorIs(err, ErrFieldDeleted)
}
