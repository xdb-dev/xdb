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

func (s *KVStoreTestSuite) TestRecords() {
	ctx := context.Background()
	records := tests.FakePosts(10)
	keys := x.Keys(records...)

	s.Run("PutRecords", func() {
		err := s.kv.PutRecords(ctx, records)
		s.Require().NoError(err)
	})

	s.Run("GetRecords", func() {
		got, missing, err := s.kv.GetRecords(ctx, keys)
		s.Require().NoError(err)
		s.Len(missing, 0)
		tests.AssertEqualRecords(s.T(), records, got)
	})

	s.Run("GetRecordsSomeMissing", func() {
		notFound := []*types.Key{
			types.NewKey("Post", "1", "not_found"),
			types.NewKey("Post", "2", "not_found"),
		}

		got, missing, err := s.kv.GetRecords(ctx, append(keys, notFound...))
		s.Require().NoError(err)
		tests.AssertEqualKeys(s.T(), notFound, missing)
		tests.AssertEqualRecords(s.T(), records, got)
	})

	s.Run("DeleteRecords", func() {
		err := s.kv.DeleteRecords(ctx, keys)
		s.Require().NoError(err)
	})

	s.Run("GetRecordsAllMissing", func() {
		got, missing, err := s.kv.GetRecords(ctx, keys)
		s.Require().NoError(err)
		s.NotEmpty(missing)
		s.Len(got, 0)
		tests.AssertEqualKeys(s.T(), missing, keys)
	})
}
