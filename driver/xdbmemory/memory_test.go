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

func TestMemoryDriver_Tuples(t *testing.T) {
	t.Parallel()
	driver := New()

	tests.TestTupleReaderWriter(t, driver)
}

func TestMemoryDriver_Records(t *testing.T) {
	t.Parallel()

	driver := New()
	tests.TestRecordReaderWriter(t, driver)
}
