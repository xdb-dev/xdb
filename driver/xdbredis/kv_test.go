package xdbredis_test

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/driver/xdbredis"
	"github.com/xdb-dev/xdb/tests"
)

type KVStoreTestSuite struct {
	suite.Suite
	db *redis.Client
	kv *xdbredis.KVStore
}

func TestKVStoreTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KVStoreTestSuite))
}

func (s *KVStoreTestSuite) SetupSuite() {
	db := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	s.db = db
	s.kv = xdbredis.New(db)
}

func (s *KVStoreTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *KVStoreTestSuite) TestTuples() {
	tuplesSuite := tests.NewTupleDriverTestSuite(s.kv)
	tuplesSuite.Basic(s.T())
}

func (s *KVStoreTestSuite) TestRecords() {
	recordsSuite := tests.NewRecordDriverTestSuite(s.kv)
	recordsSuite.Basic(s.T())
}
