package xdbredis_test

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/tests"
)

type TupleStoreTestSuite struct {
	suite.Suite
	*tests.TupleStoreTestSuite
	db *redis.Client
}

func TestTupleStoreTestSuite(t *testing.T) {
	suite.Run(t, new(TupleStoreTestSuite))
}

func (s *TupleStoreTestSuite) SetupTest() {
	s.db = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	s.db.FlushDB(s.T().Context())

	store, err := xdbredis.NewStore(s.db)
	require.NoError(s.T(), err)

	s.T().Cleanup(func() {
		_ = store.Close()
	})

	s.TupleStoreTestSuite = tests.NewTupleStoreTestSuite(store)
}

func (s *TupleStoreTestSuite) TestBasic() {
	s.TupleStoreTestSuite.Basic(s.T())
}

func (s *TupleStoreTestSuite) TestValidationStrict() {
	s.TupleStoreTestSuite.ValidationStrict(s.T())
}

func (s *TupleStoreTestSuite) TestValidationFlexible() {
	s.TupleStoreTestSuite.ValidationFlexible(s.T())
}

func (s *TupleStoreTestSuite) TestValidationDynamic() {
	s.TupleStoreTestSuite.ValidationDynamic(s.T())
}
