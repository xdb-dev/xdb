package xdbredis_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/xdb-dev/xdb/driver/xdbredis"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/x"
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
	ctx := context.Background()
	tuples := tests.FakeTuples()
	keys := x.Keys(tuples...)

	s.Run("PutTuples", func() {
		err := s.kv.PutTuples(ctx, tuples)
		s.NoError(err)
	})

	s.Run("GetTuples", func() {
		got, err := s.kv.GetTuples(ctx, keys)
		s.NoError(err)
		tests.AssertEqualTuples(s.T(), tuples, got)
	})

	s.Run("DeleteTuples", func() {
		err := s.kv.DeleteTuples(ctx, keys)
		s.NoError(err)
	})

	s.Run("GetTuplesAfterDelete", func() {

	})
}
