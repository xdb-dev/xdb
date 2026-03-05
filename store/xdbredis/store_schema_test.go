package xdbredis_test

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/store/xdbredis"
	"github.com/xdb-dev/xdb/tests"
)

type SchemaStoreTestSuite struct {
	suite.Suite
	*tests.SchemaStoreTestSuite
	db *redis.Client
}

func TestSchemaStoreTestSuite(t *testing.T) {
	suite.Run(t, new(SchemaStoreTestSuite))
}

func (s *SchemaStoreTestSuite) SetupTest() {
	s.db = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	s.db.FlushDB(s.T().Context())

	store, err := xdbredis.NewStore(s.db)
	require.NoError(s.T(), err)

	s.T().Cleanup(func() {
		_ = store.Close()
	})

	s.SchemaStoreTestSuite = tests.NewSchemaStoreTestSuite(store)
}

func (s *SchemaStoreTestSuite) TestBasic() {
	s.SchemaStoreTestSuite.Basic(s.T())
}

func (s *SchemaStoreTestSuite) TestListSchemas() {
	s.SchemaStoreTestSuite.ListSchemas(s.T())
}

func (s *SchemaStoreTestSuite) TestListNamespaces() {
	s.SchemaStoreTestSuite.ListNamespaces(s.T())
}

func (s *SchemaStoreTestSuite) TestAddNewFields() {
	s.SchemaStoreTestSuite.AddNewFields(s.T())
}

func (s *SchemaStoreTestSuite) TestDropFields() {
	s.SchemaStoreTestSuite.DropFields(s.T())
}

func (s *SchemaStoreTestSuite) TestModifyFields() {
	s.SchemaStoreTestSuite.ModifyFields(s.T())
}

func (s *SchemaStoreTestSuite) TestEdgeCases() {
	s.SchemaStoreTestSuite.EdgeCases(s.T())
}
