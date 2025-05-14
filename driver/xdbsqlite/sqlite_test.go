package xdbsqlite_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
)

type SQLiteTestSuite struct {
	suite.Suite
	db     *sql.DB
	sqlite *xdbsqlite.SQLStore
}

func TestSQLiteTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SQLiteTestSuite))
}

func (s *SQLiteTestSuite) SetupSuite() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)
	s.db = db
	s.sqlite = xdbsqlite.NewSQLStore(db)
}

func (s *SQLiteTestSuite) SetupTest() {
	schema, err := os.ReadFile("../../tests/schema.sql")
	s.Require().NoError(err)
	_, err = s.db.Exec(string(schema))
	s.Require().NoError(err)
}

func (s *SQLiteTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *SQLiteTestSuite) TestTuples() {
	ctx := context.Background()
	tuples := tests.FakeTuples()

	s.Run("PutTuples", func() {
		err := s.sqlite.PutTuples(ctx, tuples)
		s.Require().NoError(err)
	})

}
