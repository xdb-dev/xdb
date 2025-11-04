package xdbsqlite_test

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
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
	defer tx.Rollback()

	migrator := xdbsqlite.NewMigrator(tx)

	err = migrator.CreateTable(s.T().Context(), tests.FakePostSchema())
	s.Require().NoError(err)

	rows, err := tx.Query(`SELECT * FROM "com.example.Post"`)
	s.Require().NoError(err)
	defer rows.Close()

	gotColumns, err := rows.Columns()
	s.Require().NoError(err)

	expectedColumns := []string{
		"title",
		"content",
		"tags",
		"metadata",
		"rating",
		"published",
		"comments.count",
		"thumbnail",
		"created_at",
	}

	s.Equal(expectedColumns, gotColumns)
}

func (s *MigratorTestSuite) TestAlterTable_NoChange() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := xdbsqlite.NewMigrator(tx)

	schema := tests.FakePostSchema()

	err = migrator.CreateTable(s.T().Context(), schema)
	s.Require().NoError(err)

	err = migrator.AlterTable(s.T().Context(), schema, schema)
	s.Require().NoError(err)

	rows, err := tx.Query(`SELECT * FROM "com.example.Post"`)
	s.Require().NoError(err)
	defer rows.Close()

	gotColumns, err := rows.Columns()
	s.Require().NoError(err)

	expectedColumns := []string{
		"title",
		"content",
		"tags",
		"metadata",
		"rating",
		"published",
		"comments.count",
		"thumbnail",
		"created_at",
	}

	s.Equal(expectedColumns, gotColumns)
}

func (s *MigratorTestSuite) TestAlterTable_AddFields() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := xdbsqlite.NewMigrator(tx)

	schema := tests.FakePostSchema()

	err = migrator.CreateTable(s.T().Context(), schema)
	s.Require().NoError(err)

	extendedSchema := tests.FakePostSchema()
	extendedSchema.Fields = append(schema.Fields, &core.Schema{
		Name: "author",
		Type: core.TypeIDString.String(),
	})

	err = migrator.AlterTable(s.T().Context(), schema, extendedSchema)
	s.Require().NoError(err)

	rows, err := tx.Query(`SELECT * FROM "com.example.Post"`)
	s.Require().NoError(err)
	defer rows.Close()

	gotColumns, err := rows.Columns()
	s.Require().NoError(err)

	expectedColumns := []string{
		"title",
		"content",
		"tags",
		"metadata",
		"rating",
		"published",
		"comments.count",
		"thumbnail",
		"created_at",
		"author",
	}

	s.Equal(expectedColumns, gotColumns)
}

func (s *MigratorTestSuite) TestAlterTable_DropFields() {
	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := xdbsqlite.NewMigrator(tx)

	schema := tests.FakePostSchema()

	err = migrator.CreateTable(s.T().Context(), schema)
	s.Require().NoError(err)

	reducedSchema := tests.FakePostSchema()
	reducedSchema.Fields = reducedSchema.Fields[:len(reducedSchema.Fields)-1]

	err = migrator.AlterTable(s.T().Context(), schema, reducedSchema)
	s.ErrorIs(err, xdbsqlite.ErrFieldDeleted)
}

func (s *MigratorTestSuite) TestAllTypeMappings() {
	// Test schema with one field of each supported type
	allTypesSchema := &core.Schema{
		Name: "com.example.AllTypes",
		Fields: []*core.Schema{
			{Name: "field_string", Type: core.TypeIDString.String()},
			{Name: "field_integer", Type: core.TypeIDInteger.String()},
			{Name: "field_boolean", Type: core.TypeIDBoolean.String()},
			{Name: "field_time", Type: core.TypeIDTime.String()},
			{Name: "field_float", Type: core.TypeIDFloat.String()},
			{Name: "field_bytes", Type: core.TypeIDBytes.String()},
			{Name: "field_array", Type: core.TypeIDArray.String()},
			{Name: "field_map", Type: core.TypeIDMap.String()},
		},
	}

	tx, err := s.db.BeginTx(s.T().Context(), nil)
	s.Require().NoError(err)
	defer tx.Rollback()

	migrator := xdbsqlite.NewMigrator(tx)

	err = migrator.CreateTable(s.T().Context(), allTypesSchema)
	s.Require().NoError(err)

	rows, err := tx.Query(`SELECT * FROM "com.example.AllTypes"`)
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
