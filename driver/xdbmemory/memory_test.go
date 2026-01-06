package xdbmemory

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/tests"
)

type MemoryDriverTestSuite struct {
	suite.Suite
	*tests.SchemaDriverTestSuite
}

func TestMemoryDriverTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryDriverTestSuite))
}

func (s *MemoryDriverTestSuite) SetupTest() {
	driver := New()
	s.SchemaDriverTestSuite = tests.NewSchemaDriverTestSuite(driver)
}

func (s *MemoryDriverTestSuite) TestBasic() {
	s.SchemaDriverTestSuite.Basic(s.T())
}

func (s *MemoryDriverTestSuite) TestListSchemas() {
	s.SchemaDriverTestSuite.ListSchemas(s.T())
}

func (s *MemoryDriverTestSuite) TestAddNewFields() {
	s.SchemaDriverTestSuite.AddNewFields(s.T())
}

func (s *MemoryDriverTestSuite) TestDropFields() {
	s.SchemaDriverTestSuite.DropFields(s.T())
}

func (s *MemoryDriverTestSuite) TestModifyFields() {
	s.SchemaDriverTestSuite.ModifyFields(s.T())
}

func (s *MemoryDriverTestSuite) TestEdgeCases() {
	s.SchemaDriverTestSuite.EdgeCases(s.T())
}

type TupleDriverTestSuite struct {
	suite.Suite
	*tests.TupleDriverTestSuite
}

func TestTupleDriverTestSuite(t *testing.T) {
	suite.Run(t, new(TupleDriverTestSuite))
}

func (s *TupleDriverTestSuite) SetupTest() {
	driver := New()
	s.TupleDriverTestSuite = tests.NewTupleDriverTestSuite(driver)
}

func (s *TupleDriverTestSuite) TestBasic() {
	s.TupleDriverTestSuite.Basic(s.T())
}

func (s *TupleDriverTestSuite) TestValidationStrict() {
	s.TupleDriverTestSuite.ValidationStrict(s.T())
}

func (s *TupleDriverTestSuite) TestValidationFlexible() {
	s.TupleDriverTestSuite.ValidationFlexible(s.T())
}

func (s *TupleDriverTestSuite) TestValidationDynamic() {
	s.TupleDriverTestSuite.ValidationDynamic(s.T())
}

type RecordDriverTestSuite struct {
	suite.Suite
	*tests.RecordDriverTestSuite
}

func TestRecordDriverTestSuite(t *testing.T) {
	suite.Run(t, new(RecordDriverTestSuite))
}

func (s *RecordDriverTestSuite) SetupTest() {
	driver := New()
	s.RecordDriverTestSuite = tests.NewRecordDriverTestSuite(driver)
}

func (s *RecordDriverTestSuite) TestBasic() {
	s.RecordDriverTestSuite.Basic(s.T())
}

func (s *RecordDriverTestSuite) TestValidationStrict() {
	s.RecordDriverTestSuite.ValidationStrict(s.T())
}

func (s *RecordDriverTestSuite) TestValidationFlexible() {
	s.RecordDriverTestSuite.ValidationFlexible(s.T())
}

func (s *RecordDriverTestSuite) TestValidationDynamic() {
	s.RecordDriverTestSuite.ValidationDynamic(s.T())
}
