package xdbsqlite_test

import (
	"testing"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/store/xdbsqlite"
	"github.com/xdb-dev/xdb/tests"
)

type RecordStoreTestSuite struct {
	suite.Suite
	*tests.RecordStoreTestSuite
}

func TestRecordStoreTestSuite(t *testing.T) {
	suite.Run(t, new(RecordStoreTestSuite))
}

func (s *RecordStoreTestSuite) SetupTest() {
	cfg := xdbsqlite.Config{
		InMemory: true,
	}

	store, err := xdbsqlite.New(cfg)
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
