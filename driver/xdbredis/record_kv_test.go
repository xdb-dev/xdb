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

type RecordKVStoreTestSuite struct {
	suite.Suite
	db *redis.Client
	kv *xdbredis.RecordKVStore
}

func TestRecordKVStoreTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RecordKVStoreTestSuite))
}

func (s *RecordKVStoreTestSuite) SetupSuite() {
	db := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	s.db = db
	s.kv = xdbredis.NewRecordKVStore(db)
}

func (s *RecordKVStoreTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *RecordKVStoreTestSuite) TestRecords() {
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
			types.NewKey("Test", "1"),
			types.NewKey("Test", "2"),
		}

		got, missing, err := s.kv.GetRecords(ctx, append(keys, notFound...))
		s.Require().NoError(err)
		s.NotEmpty(missing)
		tests.AssertEqualKeys(s.T(), missing, notFound)
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
