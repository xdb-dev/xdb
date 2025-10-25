package xdbsqlite_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

type MigratorTestSuite struct {
	suite.Suite
	db       *sql.DB
	tx       *sql.Tx
	migrator *xdbsqlite.Migrator
	schema   *core.Schema
}

func TestMigratorSuite(t *testing.T) {
	suite.Run(t, new(MigratorTestSuite))
}

func (s *MigratorTestSuite) SetupSuite() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)
	s.db = db

	s.schema = &core.Schema{
		Name: "com.example.Post",
		Fields: []*core.Schema{
			{
				Name:     "title",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
			{
				Name:     "content",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
			{
				Name: "tags",
				Type: core.TypeIDArray.String(),
				Items: &core.Schema{
					Type: core.TypeIDString.String(),
				},
			},
			{
				Name: "metadata",
				Type: core.TypeIDMap.String(),
			},
			{
				Name: "rating",
				Type: core.TypeIDFloat.String(),
			},
			{
				Name: "published",
				Type: core.TypeIDBoolean.String(),
			},
			{
				Name: "comments.count",
				Type: core.TypeIDInteger.String(),
			},
			{
				Name: "thumbnail",
				Type: core.TypeIDBytes.String(),
			},
			{
				Name: "created_at",
				Type: core.TypeIDTime.String(),
			},
		},
	}
}

func (s *MigratorTestSuite) SetupTest() {
	tx, err := s.db.BeginTx(context.Background(), nil)
	s.Require().NoError(err)
	s.tx = tx
	s.migrator = xdbsqlite.NewMigrator(tx)
}

func (s *MigratorTestSuite) TearDownTest() {
	if s.tx != nil {
		s.tx.Rollback()
	}
}

func (s *MigratorTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *MigratorTestSuite) TestGenerateCreateTable() {
	up, down, err := s.migrator.Generate(context.Background(), nil, s.schema)
	s.Require().NoError(err)

	expectedUp := normalize(`CREATE TABLE IF NOT EXISTS "com.example.Post" (
		"title" TEXT NOT NULL,
		"content" TEXT NOT NULL,
		"tags" TEXT,
		"metadata" TEXT,
		"rating" REAL,
		"published" INTEGER,
		"comments.count" INTEGER,
		"thumbnail" BLOB,
		"created_at" INTEGER
	);`)
	expectedDown := normalize(`DROP TABLE IF EXISTS "com.example.Post";`)

	s.Equal(expectedUp, normalize(up))
	s.Equal(expectedDown, normalize(down))

	// Verify all SQLite type mappings are present
	s.Contains(up, "TEXT")      // String type
	s.Contains(up, "REAL")      // Float type
	s.Contains(up, "INTEGER")   // Integer, Boolean, Time types
	s.Contains(up, "BLOB")      // Bytes type
	s.Contains(up, "NOT NULL")  // Required constraint

	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	_, err = s.tx.Exec(down)
	s.Require().NoError(err)
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

	up, _, err := s.migrator.Generate(context.Background(), nil, allTypesSchema)
	s.Require().NoError(err)

	// Verify each type maps correctly
	s.Contains(up, `"field_string" TEXT`)
	s.Contains(up, `"field_integer" INTEGER`)
	s.Contains(up, `"field_boolean" INTEGER`)
	s.Contains(up, `"field_time" INTEGER`)
	s.Contains(up, `"field_float" REAL`)
	s.Contains(up, `"field_bytes" BLOB`)
	s.Contains(up, `"field_array" TEXT`)
	s.Contains(up, `"field_map" TEXT`)

	// Verify the table can be created
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)
}

func (s *MigratorTestSuite) TestGenerateAlterTableNoChange() {
	noChangeSchema := &core.Schema{
		Name:   "com.example.NoChange",
		Fields: s.schema.Fields,
	}

	// First create the table
	up, _, err := s.migrator.Generate(context.Background(), nil, noChangeSchema)
	s.Require().NoError(err)
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	// Now try to generate with no changes
	up, down, err := s.migrator.Generate(context.Background(), noChangeSchema, noChangeSchema)
	s.Require().NoError(err)
	s.Empty(up)
	s.Empty(down)
}

func (s *MigratorTestSuite) TestGenerateAlterTableAddFields() {
	addFieldsSchema := &core.Schema{
		Name:   "com.example.AddFields",
		Fields: s.schema.Fields,
	}

	// First create the table with initial schema
	up, _, err := s.migrator.Generate(context.Background(), nil, addFieldsSchema)
	s.Require().NoError(err)
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	// Now add new fields
	extendedSchema := &core.Schema{
		Name: "com.example.AddFields",
		Fields: append(addFieldsSchema.Fields, []*core.Schema{
			{
				Name: "author",
				Type: core.TypeIDString.String(),
			},
			{
				Name: "views",
				Type: core.TypeIDInteger.String(),
			},
		}...),
	}

	up, down, err := s.migrator.Generate(context.Background(), addFieldsSchema, extendedSchema)
	s.Require().NoError(err)

	expectedUp := normalize(`ALTER TABLE "com.example.AddFields" ADD COLUMN "author" TEXT;
	ALTER TABLE "com.example.AddFields" ADD COLUMN "views" INTEGER;`)
	expectedDown := normalize(`ALTER TABLE "com.example.AddFields" DROP COLUMN "author";
	ALTER TABLE "com.example.AddFields" DROP COLUMN "views";`)

	s.Equal(expectedUp, normalize(up))
	s.Equal(expectedDown, normalize(down))

	// Execute the migration
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	// Verify columns were added
	rows, err := s.tx.Query(`PRAGMA table_info("com.example.AddFields")`)
	s.Require().NoError(err)
	defer rows.Close()

	columnNames := []string{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		s.Require().NoError(err)
		columnNames = append(columnNames, name)
	}

	s.Contains(columnNames, "author")
	s.Contains(columnNames, "views")
}

func (s *MigratorTestSuite) TestGenerateAlterTableDropFieldsError() {
	dropFieldsSchema := &core.Schema{
		Name:   "com.example.DropFields",
		Fields: s.schema.Fields,
	}

	// First create the table
	up, _, err := s.migrator.Generate(context.Background(), nil, dropFieldsSchema)
	s.Require().NoError(err)
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	// Try to drop fields
	reducedSchema := &core.Schema{
		Name: "com.example.DropFields",
		Fields: []*core.Schema{
			{
				Name:     "title",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
		},
	}

	_, _, err = s.migrator.Generate(context.Background(), dropFieldsSchema, reducedSchema)
	s.Require().Error(err)
	s.ErrorIs(err, xdbsqlite.ErrFieldDeleted)
}

func (s *MigratorTestSuite) TestGenerateAlterTableFieldTypeChangeError() {
	typeChangeSchema := &core.Schema{
		Name:   "com.example.TypeChange",
		Fields: s.schema.Fields,
	}

	// First create the table
	up, _, err := s.migrator.Generate(context.Background(), nil, typeChangeSchema)
	s.Require().NoError(err)
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	// Try to change field type
	modifiedSchema := &core.Schema{
		Name: "com.example.TypeChange",
		Fields: []*core.Schema{
			{
				Name:     "title",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
			{
				Name:     "content",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
			{
				Name: "tags",
				Type: core.TypeIDString.String(), // Changed from Array to String
			},
			{
				Name: "metadata",
				Type: core.TypeIDMap.String(),
			},
			{
				Name: "rating",
				Type: core.TypeIDFloat.String(),
			},
			{
				Name: "published",
				Type: core.TypeIDBoolean.String(),
			},
			{
				Name: "comments.count",
				Type: core.TypeIDInteger.String(),
			},
			{
				Name: "thumbnail",
				Type: core.TypeIDBytes.String(),
			},
			{
				Name: "created_at",
				Type: core.TypeIDTime.String(),
			},
		},
	}

	_, _, err = s.migrator.Generate(context.Background(), typeChangeSchema, modifiedSchema)
	s.Require().Error(err)
	s.ErrorIs(err, xdbsqlite.ErrFieldModified)
}

func (s *MigratorTestSuite) TestGenerateAlterTableFieldConstraintChangeError() {
	constraintChangeSchema := &core.Schema{
		Name:   "com.example.ConstraintChange",
		Fields: s.schema.Fields,
	}

	// First create the table
	up, _, err := s.migrator.Generate(context.Background(), nil, constraintChangeSchema)
	s.Require().NoError(err)
	_, err = s.tx.Exec(up)
	s.Require().NoError(err)

	// Try to change field constraint
	modifiedSchema := &core.Schema{
		Name: "com.example.ConstraintChange",
		Fields: []*core.Schema{
			{
				Name:     "title",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
			{
				Name:     "content",
				Type:     core.TypeIDString.String(),
				Required: true,
			},
			{
				Name: "tags",
				Type: core.TypeIDArray.String(),
				Items: &core.Schema{
					Type: core.TypeIDString.String(),
				},
				Required: true, // Changed from false to true
			},
			{
				Name: "metadata",
				Type: core.TypeIDMap.String(),
			},
			{
				Name: "rating",
				Type: core.TypeIDFloat.String(),
			},
			{
				Name: "published",
				Type: core.TypeIDBoolean.String(),
			},
			{
				Name: "comments.count",
				Type: core.TypeIDInteger.String(),
			},
			{
				Name: "thumbnail",
				Type: core.TypeIDBytes.String(),
			},
			{
				Name: "created_at",
				Type: core.TypeIDTime.String(),
			},
		},
	}

	_, _, err = s.migrator.Generate(context.Background(), constraintChangeSchema, modifiedSchema)
	s.Require().Error(err)
	s.ErrorIs(err, xdbsqlite.ErrFieldModified)
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\\n", "")
	s = strings.ReplaceAll(s, "\t", "  ")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, " ", "")

	return s
}
