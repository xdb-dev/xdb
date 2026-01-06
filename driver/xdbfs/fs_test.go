package xdbfs_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/driver/xdbfs"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/tests"
)

type FSDriverTestSuite struct {
	suite.Suite
	*tests.SchemaDriverTestSuite
	tmpDir string
}

func TestFSDriverTestSuite(t *testing.T) {
	suite.Run(t, new(FSDriverTestSuite))
}

func (s *FSDriverTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	driver, err := xdbfs.New(tmpDir, xdbfs.WithSharedAccess())
	require.NoError(s.T(), err)

	s.tmpDir = tmpDir
	s.SchemaDriverTestSuite = tests.NewSchemaDriverTestSuite(driver)
}

func (s *FSDriverTestSuite) TestBasic() {
	s.SchemaDriverTestSuite.Basic(s.T())
}

func (s *FSDriverTestSuite) TestListSchemas() {
	s.SchemaDriverTestSuite.ListSchemas(s.T())
}

func (s *FSDriverTestSuite) TestAddNewFields() {
	s.SchemaDriverTestSuite.AddNewFields(s.T())
}

func (s *FSDriverTestSuite) TestDropFields() {
	s.SchemaDriverTestSuite.DropFields(s.T())
}

func (s *FSDriverTestSuite) TestModifyFields() {
	s.SchemaDriverTestSuite.ModifyFields(s.T())
}

func (s *FSDriverTestSuite) TestEdgeCases() {
	s.SchemaDriverTestSuite.EdgeCases(s.T())
}

type TupleDriverTestSuite struct {
	suite.Suite
	*tests.TupleDriverTestSuite
	tmpDir string
}

func TestTupleDriverTestSuite(t *testing.T) {
	suite.Run(t, new(TupleDriverTestSuite))
}

func (s *TupleDriverTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	driver, err := xdbfs.New(tmpDir, xdbfs.WithSharedAccess())
	require.NoError(s.T(), err)

	s.tmpDir = tmpDir
	s.TupleDriverTestSuite = tests.NewTupleDriverTestSuite(driver)
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
	tmpDir string
}

func TestRecordDriverTestSuite(t *testing.T) {
	suite.Run(t, new(RecordDriverTestSuite))
}

func (s *RecordDriverTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	driver, err := xdbfs.New(tmpDir, xdbfs.WithSharedAccess())
	require.NoError(s.T(), err)

	s.tmpDir = tmpDir
	s.RecordDriverTestSuite = tests.NewRecordDriverTestSuite(driver)
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

func TestFSDriver_Permissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		opts             []xdbfs.Option
		expectedFilePerm os.FileMode
	}{
		{
			name:             "default restrictive permissions",
			opts:             nil,
			expectedFilePerm: 0o600,
		},
		{
			name:             "shared access permissions",
			opts:             []xdbfs.Option{xdbfs.WithSharedAccess()},
			expectedFilePerm: 0o644,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir := t.TempDir()

			driver, err := xdbfs.New(tmpDir, tt.opts...)
			require.NoError(t, err)

			ctx := context.Background()
			uri, err := core.ParseURI("xdb://test/users")
			require.NoError(t, err)

			def := &schema.Def{
				Name: "users",
				Fields: []*schema.FieldDef{
					{Name: "name", Type: core.TypeString},
				},
			}

			err = driver.PutSchema(ctx, uri, def)
			require.NoError(t, err)

			schemaFile := tmpDir + "/test/users/.schema.json"

			fileInfo, err := os.Stat(schemaFile)
			require.NoError(t, err)
			require.False(t, fileInfo.IsDir())
			require.Equal(t, tt.expectedFilePerm, fileInfo.Mode().Perm(), "file permissions mismatch")

			record := core.NewRecord("test", "users", "user1")
			record.Set("name", "Alice")
			err = driver.PutRecords(ctx, []*core.Record{record})
			require.NoError(t, err)

			recordFile := tmpDir + "/test/users/user1.json"
			recordFileInfo, err := os.Stat(recordFile)
			require.NoError(t, err)
			require.Equal(t, tt.expectedFilePerm, recordFileInfo.Mode().Perm(), "record file permissions mismatch")
		})
	}
}
