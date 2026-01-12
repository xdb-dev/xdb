package xdbfs_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/store/xdbfs"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/tests"
)

type FSStoreTestSuite struct {
	suite.Suite
	*tests.SchemaStoreTestSuite
	tmpDir string
}

func TestFSStoreTestSuite(t *testing.T) {
	suite.Run(t, new(FSStoreTestSuite))
}

func (s *FSStoreTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	driver, err := xdbfs.New(tmpDir, xdbfs.WithSharedAccess())
	require.NoError(s.T(), err)

	s.tmpDir = tmpDir
	s.SchemaStoreTestSuite = tests.NewSchemaStoreTestSuite(driver)
}

func (s *FSStoreTestSuite) TestBasic() {
	s.SchemaStoreTestSuite.Basic(s.T())
}

func (s *FSStoreTestSuite) TestListSchemas() {
	s.SchemaStoreTestSuite.ListSchemas(s.T())
}

func (s *FSStoreTestSuite) TestAddNewFields() {
	s.SchemaStoreTestSuite.AddNewFields(s.T())
}

func (s *FSStoreTestSuite) TestDropFields() {
	s.SchemaStoreTestSuite.DropFields(s.T())
}

func (s *FSStoreTestSuite) TestModifyFields() {
	s.SchemaStoreTestSuite.ModifyFields(s.T())
}

func (s *FSStoreTestSuite) TestEdgeCases() {
	s.SchemaStoreTestSuite.EdgeCases(s.T())
}

type TupleStoreTestSuite struct {
	suite.Suite
	*tests.TupleStoreTestSuite
	tmpDir string
}

func TestTupleStoreTestSuite(t *testing.T) {
	suite.Run(t, new(TupleStoreTestSuite))
}

func (s *TupleStoreTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	driver, err := xdbfs.New(tmpDir, xdbfs.WithSharedAccess())
	require.NoError(s.T(), err)

	s.tmpDir = tmpDir
	s.TupleStoreTestSuite = tests.NewTupleStoreTestSuite(driver)
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
	tmpDir string
}

func TestRecordStoreTestSuite(t *testing.T) {
	suite.Run(t, new(RecordStoreTestSuite))
}

func (s *RecordStoreTestSuite) SetupTest() {
	tmpDir := s.T().TempDir()
	driver, err := xdbfs.New(tmpDir, xdbfs.WithSharedAccess())
	require.NoError(s.T(), err)

	s.tmpDir = tmpDir
	s.RecordStoreTestSuite = tests.NewRecordStoreTestSuite(driver)
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
				NS:   uri.NS(),
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
