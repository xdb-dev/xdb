package xdbbadger_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/suite"

	"github.com/xdb-dev/xdb/driver/xdbbadger"
	"github.com/xdb-dev/xdb/tests"
)

type KVStoreTestSuite struct {
	suite.Suite
	db      *badger.DB
	kv      *xdbbadger.KVStore
	tempDir string
}

func TestKVStoreTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KVStoreTestSuite))
}

func (s *KVStoreTestSuite) SetupSuite() {
	// Create temporary directory for BadgerDB
	tempDir, err := os.MkdirTemp("", "xdbbadger_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Configure BadgerDB options
	opts := badger.DefaultOptions(filepath.Join(tempDir, "badger.db"))
	opts.Logger = nil // Disable logging for tests

	// Open BadgerDB
	db, err := badger.Open(opts)
	s.Require().NoError(err)

	s.db = db
	s.kv = xdbbadger.New(db)
}

func (s *KVStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func (s *KVStoreTestSuite) TestTuples() {
	tests.TestTupleReaderWriter(s.T(), s.kv)
}

func (s *KVStoreTestSuite) TestRecords() {
	tests.TestRecordReaderWriter(s.T(), s.kv)
}
