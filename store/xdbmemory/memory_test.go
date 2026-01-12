package xdbmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/tests"
)

type MemoryStoreTestSuite struct {
	suite.Suite
	*tests.SchemaStoreTestSuite
}

func TestMemoryStoreTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryStoreTestSuite))
}

func (s *MemoryStoreTestSuite) SetupTest() {
	driver := New()
	s.SchemaStoreTestSuite = tests.NewSchemaStoreTestSuite(driver)
}

func (s *MemoryStoreTestSuite) TestBasic() {
	s.SchemaStoreTestSuite.Basic(s.T())
}

func (s *MemoryStoreTestSuite) TestListSchemas() {
	s.SchemaStoreTestSuite.ListSchemas(s.T())
}

func (s *MemoryStoreTestSuite) TestListNamespaces() {
	s.SchemaStoreTestSuite.ListNamespaces(s.T())
}

func (s *MemoryStoreTestSuite) TestAddNewFields() {
	s.SchemaStoreTestSuite.AddNewFields(s.T())
}

func (s *MemoryStoreTestSuite) TestDropFields() {
	s.SchemaStoreTestSuite.DropFields(s.T())
}

func (s *MemoryStoreTestSuite) TestModifyFields() {
	s.SchemaStoreTestSuite.ModifyFields(s.T())
}

func (s *MemoryStoreTestSuite) TestEdgeCases() {
	s.SchemaStoreTestSuite.EdgeCases(s.T())
}

type TupleStoreTestSuite struct {
	suite.Suite
	*tests.TupleStoreTestSuite
}

func TestTupleStoreTestSuite(t *testing.T) {
	suite.Run(t, new(TupleStoreTestSuite))
}

func (s *TupleStoreTestSuite) SetupTest() {
	driver := New()
	s.TupleStoreTestSuite = tests.NewTupleStoreTestSuite(driver)
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

type RecordStoreTestSuite struct {
	suite.Suite
	*tests.RecordStoreTestSuite
}

func TestRecordStoreTestSuite(t *testing.T) {
	suite.Run(t, new(RecordStoreTestSuite))
}

func (s *RecordStoreTestSuite) SetupTest() {
	driver := New()
	s.RecordStoreTestSuite = tests.NewRecordStoreTestSuite(driver)
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

func TestHealth(t *testing.T) {
	store := New()
	ctx := context.Background()

	err := store.Health(ctx)
	require.NoError(t, err)

	err = store.Health(ctx)
	require.NoError(t, err)

	err = store.Health(ctx)
	require.NoError(t, err)
}
