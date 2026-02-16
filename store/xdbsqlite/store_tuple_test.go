package xdbsqlite_test

import (
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
)

type TupleStoreTestSuite struct {
	suite.Suite
	*tests.TupleStoreTestSuite
}

func TestTupleStoreTestSuite(t *testing.T) {
	suite.Run(t, new(TupleStoreTestSuite))
}

func (s *TupleStoreTestSuite) SetupTest() {
	cfg := xdbsqlite.Config{
		InMemory: true,
	}

	store, err := xdbsqlite.New(cfg)
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
