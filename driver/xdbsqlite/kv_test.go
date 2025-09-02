package xdbsqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/driver/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
)

type KVStoreTestSuite struct {
	suite.Suite
	db *sql.DB
	kv *xdbsqlite.KVStore
}

func TestKVStoreTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KVStoreTestSuite))
}

func (s *KVStoreTestSuite) SetupSuite() {
	db, err := sql.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	s.db = db
	s.kv = xdbsqlite.NewKVStore(db)

	err = s.kv.Migrate(context.Background(), []string{
		"Test",
		"Post",
	})
	s.Require().NoError(err)
}

func (s *KVStoreTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *KVStoreTestSuite) TestTuples() {
	tests.TestTupleReaderWriter(s.T(), s.kv)
}

func (s *KVStoreTestSuite) TestRecords() {
	tests.TestRecordReaderWriter(s.T(), s.kv)
}
