package xdbsqlite_test

import (
	"context"
	"database/sql"
	"testing"

	sqlite3driver "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/driver/xdbsqlite"
	"github.com/xdb-dev/xdb/driver/xdbsqlite/internal"
	"github.com/xdb-dev/xdb/tests"
)

type SQLiteDriverTestSuite struct {
	suite.Suite
	*tests.SchemaDriverTestSuite
	db *sql.DB
	tx *sql.Tx
}

func TestSQLiteDriverTestSuite(t *testing.T) {
	suite.Run(t, new(SQLiteDriverTestSuite))
}

func (s *SQLiteDriverTestSuite) SetupTest() {
	db, err := sqlite3driver.Open(":memory:")
	require.NoError(s.T(), err)

	tx, err := db.Begin()
	require.NoError(s.T(), err)

	queries := internal.NewQueries(tx)
	err = queries.CreateMetadataTable(context.Background())
	require.NoError(s.T(), err)

	s.db = db
	s.tx = tx
	s.SchemaDriverTestSuite = tests.NewSchemaDriverTestSuite(xdbsqlite.NewSchemaDriverTx(tx))
}

func (s *SQLiteDriverTestSuite) TearDownTest() {
	if s.tx != nil {
		s.tx.Rollback()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *SQLiteDriverTestSuite) TestBasic() {
	s.SchemaDriverTestSuite.Basic(s.T())
}

func (s *SQLiteDriverTestSuite) TestListSchemas() {
	s.SchemaDriverTestSuite.ListSchemas(s.T())
}

func (s *SQLiteDriverTestSuite) TestAddNewFields() {
	s.SchemaDriverTestSuite.AddNewFields(s.T())
}

func (s *SQLiteDriverTestSuite) TestDropFields() {
	s.SchemaDriverTestSuite.DropFields(s.T())
}

func (s *SQLiteDriverTestSuite) TestModifyFields() {
	s.SchemaDriverTestSuite.ModifyFields(s.T())
}

func (s *SQLiteDriverTestSuite) TestEdgeCases() {
	s.SchemaDriverTestSuite.EdgeCases(s.T())
}
