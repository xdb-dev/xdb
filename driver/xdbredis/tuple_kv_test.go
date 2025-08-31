package xdbredis_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/driver/xdbredis"
	"github.com/xdb-dev/xdb/tests"
	"github.com/xdb-dev/xdb/types"
	"github.com/xdb-dev/xdb/x"
)

type TupleKVStoreTestSuite struct {
	suite.Suite
	db *redis.Client
	kv *xdbredis.TupleKVStore
}

func TestKVStoreTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TupleKVStoreTestSuite))
}

func (s *TupleKVStoreTestSuite) SetupSuite() {
	db := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	s.db = db
	s.kv = xdbredis.NewTupleKVStore(db)
}

func (s *TupleKVStoreTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *TupleKVStoreTestSuite) TestTuples() {
	ctx := context.Background()
	tuples := tests.FakeTuples()
	keys := x.Keys(tuples...)

	s.Run("PutTuples", func() {
		err := s.kv.PutTuples(ctx, tuples)
		s.Require().NoError(err)
	})

	s.Run("GetTuples", func() {
		got, missing, err := s.kv.GetTuples(ctx, keys)
		s.Require().NoError(err)
		s.Len(missing, 0)
		tests.AssertEqualTuples(s.T(), tuples, got)
	})

	s.Run("GetTuplesSomeMissing", func() {
		notFound := []*types.Key{
			types.NewKey("Test", "1", "not_found"),
			types.NewKey("Test", "2", "not_found"),
		}

		got, missing, err := s.kv.GetTuples(ctx, append(keys, notFound...))
		s.Require().NoError(err)
		s.NotEmpty(missing)
		tests.AssertEqualKeys(s.T(), missing, notFound)
		tests.AssertEqualTuples(s.T(), tuples, got)
	})

	s.Run("DeleteTuples", func() {
		err := s.kv.DeleteTuples(ctx, keys)
		s.Require().NoError(err)
	})

	s.Run("GetTuplesAllMissing", func() {
		got, missing, err := s.kv.GetTuples(ctx, keys)
		s.Require().NoError(err)
		s.NotEmpty(missing)
		s.Len(got, 0)
		tests.AssertEqualKeys(s.T(), missing, keys)
	})
}
