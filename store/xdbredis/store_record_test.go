package xdbredis_test

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/tests"
)

type RecordStoreTestSuite struct {
	suite.Suite
	*tests.RecordStoreTestSuite
	db *redis.Client
}

func TestRecordStoreTestSuite(t *testing.T) {
	suite.Run(t, new(RecordStoreTestSuite))
}

func (s *RecordStoreTestSuite) SetupTest() {
	s.db = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	s.db.FlushDB(s.T().Context())

	store, err := xdbredis.NewStore(s.db)
	require.NoError(s.T(), err)

	s.T().Cleanup(func() {
		_ = store.Close()
	})

	s.RecordStoreTestSuite = tests.NewRecordStoreTestSuite(store)
}

func (s *RecordStoreTestSuite) TestBasic() {
	s.RecordStoreTestSuite.Basic(s.T())
}

func (s *RecordStoreTestSuite) TestValidationStrict() {
	s.RecordStoreTestSuite.ValidationStrict(s.T())
}

func (s *RecordStoreTestSuite) TestValidationFlexible() {
	s.RecordStoreTestSuite.ValidationFlexible(s.T())
}

func (s *RecordStoreTestSuite) TestValidationDynamic() {
	s.RecordStoreTestSuite.ValidationDynamic(s.T())
}
