package xdbsqlite_test

// import (
// 	"context"
// 	"database/sql"
// 	"os"
// 	"testing"

// 	_ "github.com/mattn/go-sqlite3"
// 	"github.com/stretchr/testify/suite"

// 	"github.com/xdb-dev/xdb/core"
// 	"github.com/xdb-dev/xdb/driver/xdbsqlite"
// 	"github.com/xdb-dev/xdb/registry"
// 	"github.com/xdb-dev/xdb/tests"
// 	"github.com/xdb-dev/xdb/x"
// )

// type SQLiteTestSuite struct {
// 	suite.Suite
// 	db     *sql.DB
// 	sqlite *xdbsqlite.SQLStore
// }

// func TestSQLiteTestSuite(t *testing.T) {
// 	suite.Run(t, new(SQLiteTestSuite))
// }

// func (s *SQLiteTestSuite) SetupSuite() {
// 	db, err := sql.Open("sqlite3", ":memory:")
// 	s.Require().NoError(err)

// 	registry := registry.NewRegistry()
// 	registry.Add(tests.FakeTupleSchema())
// 	registry.Add(tests.FakePostSchema())

// 	s.db = db
// 	s.sqlite = xdbsqlite.NewSQLStore(db, registry)
// }

// func (s *SQLiteTestSuite) SetupTest() {
// 	schema, err := os.ReadFile("../../tests/schema.sql")
// 	s.Require().NoError(err)
// 	_, err = s.db.Exec(string(schema))
// 	s.Require().NoError(err)
// }

// func (s *SQLiteTestSuite) TearDownSuite() {
// 	s.db.Close()
// }

// func (s *SQLiteTestSuite) TestTuples() {
// 	ctx := context.Background()
// 	tuples := tests.FakeTuples()
// 	keys := x.Keys(tuples...)

// 	s.Run("PutTuples", func() {
// 		err := s.sqlite.PutTuples(ctx, tuples)
// 		s.Require().NoError(err)
// 	})

// 	s.Run("GetTuples", func() {
// 		got, _, err := s.sqlite.GetTuples(ctx, keys)
// 		s.Require().NoError(err)
// 		tests.AssertEqualTuples(s.T(), tuples, got)
// 	})

// 	s.Run("GetTuplesSomeMissing", func() {
// 		notFound := []*core.Key{
// 			core.NewKey("Test", "1", "not_found"),
// 			core.NewKey("Test", "2", "not_found"),
// 		}

// 		got, missing, err := s.sqlite.GetTuples(ctx, append(keys, notFound...))
// 		s.Require().NoError(err)
// 		s.NotEmpty(missing)
// 		tests.AssertEqualKeys(s.T(), missing, notFound)
// 		tests.AssertEqualTuples(s.T(), tuples, got)
// 	})

// 	s.Run("DeleteTuples", func() {
// 		err := s.sqlite.DeleteTuples(ctx, keys)
// 		s.Require().NoError(err)
// 	})

// 	s.Run("GetTuplesAllMissing", func() {
// 		got, missing, err := s.sqlite.GetTuples(ctx, keys)
// 		s.Require().NoError(err)
// 		s.NotEmpty(missing)
// 		s.Len(got, 0)
// 		tests.AssertEqualKeys(s.T(), missing, keys)
// 	})

// }

// func (s *SQLiteTestSuite) TestRecords() {
// 	ctx := context.Background()
// 	records := tests.FakePosts(10)
// 	keys := x.Keys(records...)

// 	s.Run("PutRecords", func() {
// 		err := s.sqlite.PutRecords(ctx, records)
// 		s.Require().NoError(err)
// 	})

// 	s.Run("GetRecords", func() {
// 		got, missing, err := s.sqlite.GetRecords(ctx, keys)
// 		s.Require().NoError(err)
// 		s.Len(missing, 0)
// 		tests.AssertEqualRecords(s.T(), records, got)
// 	})

// 	s.Run("GetRecordsSomeMissing", func() {
// 		notFound := []*core.Key{
// 			core.NewKey("Post", "1", "not_found"),
// 			core.NewKey("Post", "2", "not_found"),
// 		}

// 		got, missing, err := s.sqlite.GetRecords(ctx, append(keys, notFound...))
// 		s.Require().NoError(err)
// 		tests.AssertEqualKeys(s.T(), notFound, missing)
// 		tests.AssertEqualRecords(s.T(), records, got)
// 	})

// 	s.Run("DeleteRecords", func() {
// 		err := s.sqlite.DeleteRecords(ctx, keys)
// 		s.Require().NoError(err)
// 	})

// 	s.Run("GetRecordsAllMissing", func() {
// 		got, missing, err := s.sqlite.GetRecords(ctx, keys)
// 		s.Require().NoError(err)
// 		s.NotEmpty(missing)
// 		s.Len(got, 0)
// 		tests.AssertEqualKeys(s.T(), missing, keys)
// 	})
// }
