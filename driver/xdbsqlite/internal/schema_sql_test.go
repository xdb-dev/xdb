package internal

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlite3driver "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type QueriesTestSuite struct {
	suite.Suite
	db      *sql.DB
	tx      *sql.Tx
	queries *Queries
	ctx     context.Context
}

func (s *QueriesTestSuite) SetupTest() {
	var err error
	s.db, err = sqlite3driver.Open(":memory:")
	require.NoError(s.T(), err)

	s.tx, err = s.db.Begin()
	require.NoError(s.T(), err)

	s.queries = NewQueries(s.tx)
	s.ctx = context.Background()
}

func (s *QueriesTestSuite) TearDownTest() {
	if s.tx != nil {
		s.tx.Rollback()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func TestQueriesTestSuite(t *testing.T) {
	suite.Run(t, new(QueriesTestSuite))
}

func (s *QueriesTestSuite) TestCreateMetadataTable() {
	err := s.queries.CreateMetadataTable(s.ctx)
	assert.NoError(s.T(), err)

	var tableName string
	err = s.tx.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='_xdb_metadata'").Scan(&tableName)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "_xdb_metadata", tableName)
}

func (s *QueriesTestSuite) TestCreateMetadataTableIdempotent() {
	err := s.queries.CreateMetadataTable(s.ctx)
	require.NoError(s.T(), err)

	err = s.queries.CreateMetadataTable(s.ctx)
	assert.NoError(s.T(), err)
}

func (s *QueriesTestSuite) TestPutAndGetMetadata() {
	err := s.queries.CreateMetadataTable(s.ctx)
	require.NoError(s.T(), err)

	now := time.Now().Unix()
	params := PutMetadataParams{
		URI:       "xdb://com.example/test",
		Schema:    `{"type": "object"}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.queries.PutMetadata(s.ctx, params)
	assert.NoError(s.T(), err)

	result, err := s.queries.GetMetadata(s.ctx, "xdb://com.example/test")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.EqualValues(s.T(), params, *result)
}

func (s *QueriesTestSuite) TestPutMetadataUpsert() {
	err := s.queries.CreateMetadataTable(s.ctx)
	require.NoError(s.T(), err)

	now := time.Now().Unix()
	params := PutMetadataParams{
		URI:       "xdb://com.example/test",
		Schema:    `{"type": "object"}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.queries.PutMetadata(s.ctx, params)
	require.NoError(s.T(), err)

	later := now + 100
	updatedParams := PutMetadataParams{
		URI:       "xdb://com.example/test",
		Schema:    `{"type": "array"}`,
		CreatedAt: now,
		UpdatedAt: later,
	}

	err = s.queries.PutMetadata(s.ctx, updatedParams)
	assert.NoError(s.T(), err)

	result, err := s.queries.GetMetadata(s.ctx, "xdb://com.example/test")
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.EqualValues(s.T(), updatedParams, *result)
}

func (s *QueriesTestSuite) TestGetMetadataNotFound() {
	err := s.queries.CreateMetadataTable(s.ctx)
	require.NoError(s.T(), err)

	result, err := s.queries.GetMetadata(s.ctx, "non-existent")
	assert.Error(s.T(), err)
	assert.Equal(s.T(), sql.ErrNoRows, err)
	assert.Nil(s.T(), result)
}

func (s *QueriesTestSuite) TestDeleteMetadata() {
	err := s.queries.CreateMetadataTable(s.ctx)
	require.NoError(s.T(), err)

	now := time.Now().Unix()
	params := PutMetadataParams{
		URI:       "xdb://com.example/test",
		Schema:    `{"type": "object"}`,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.queries.PutMetadata(s.ctx, params)
	require.NoError(s.T(), err)

	err = s.queries.DeleteMetadata(s.ctx, "xdb://com.example/test")
	assert.NoError(s.T(), err)

	_, err = s.queries.GetMetadata(s.ctx, "xdb://com.example/test")
	assert.Error(s.T(), err)
	assert.Equal(s.T(), sql.ErrNoRows, err)
}

func (s *QueriesTestSuite) TestDeleteMetadataNotFound() {
	err := s.queries.CreateMetadataTable(s.ctx)
	require.NoError(s.T(), err)

	err = s.queries.DeleteMetadata(s.ctx, "non-existent")
	assert.NoError(s.T(), err)
}

func (s *QueriesTestSuite) TestCreateKVTable() {
	err := s.queries.CreateKVTable(s.ctx, "test_kv")
	assert.NoError(s.T(), err)

	var tableName string
	err = s.tx.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_kv'").Scan(&tableName)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "test_kv", tableName)

	rows, err := s.tx.Query("PRAGMA table_info(test_kv)")
	require.NoError(s.T(), err)
	defer rows.Close()

	expectedColumns := map[string]bool{
		"key":        false,
		"id":         false,
		"attr":       false,
		"value":      false,
		"updated_at": false,
	}

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString

		err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk)
		require.NoError(s.T(), err)

		if _, exists := expectedColumns[name]; exists {
			expectedColumns[name] = true
		}
	}

	for col, found := range expectedColumns {
		assert.True(s.T(), found, "Column %s should exist", col)
	}
}

func (s *QueriesTestSuite) TestCreateKVTableIdempotent() {
	err := s.queries.CreateKVTable(s.ctx, "test_kv")
	require.NoError(s.T(), err)

	err = s.queries.CreateKVTable(s.ctx, "test_kv")
	assert.NoError(s.T(), err)
}

func (s *QueriesTestSuite) TestCreateSQLTable() {
	params := CreateSQLTableParams{
		Name: "users",
		Columns: [][]string{
			{"id", "TEXT", "PRIMARY KEY"},
			{"name", "TEXT", "NOT NULL"},
			{"email", "TEXT"},
			{"age", "INTEGER"},
		},
	}

	err := s.queries.CreateSQLTable(s.ctx, params)
	assert.NoError(s.T(), err)

	var tableName string
	err = s.tx.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users'").Scan(&tableName)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), "users", tableName)

	rows, err := s.tx.Query("PRAGMA table_info(users)")
	require.NoError(s.T(), err)
	defer rows.Close()

	expectedColumns := map[string]bool{
		"id":    false,
		"name":  false,
		"email": false,
		"age":   false,
	}

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString

		err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk)
		require.NoError(s.T(), err)

		if _, exists := expectedColumns[name]; exists {
			expectedColumns[name] = true
		}
	}

	for col, found := range expectedColumns {
		assert.True(s.T(), found, "Column %s should exist", col)
	}
}

func (s *QueriesTestSuite) TestCreateSQLTableIdempotent() {
	params := CreateSQLTableParams{
		Name: "users",
		Columns: [][]string{
			{"id", "TEXT", "PRIMARY KEY"},
		},
	}

	err := s.queries.CreateSQLTable(s.ctx, params)
	require.NoError(s.T(), err)

	err = s.queries.CreateSQLTable(s.ctx, params)
	assert.NoError(s.T(), err)
}

func (s *QueriesTestSuite) TestAlterSQLTableAddAndDrop() {
	createParams := CreateSQLTableParams{
		Name: "users",
		Columns: [][]string{
			{"id", "TEXT", "PRIMARY KEY"},
			{"old_field", "TEXT"},
		},
	}

	err := s.queries.CreateSQLTable(s.ctx, createParams)
	require.NoError(s.T(), err)

	alterParams := AlterSQLTableParams{
		Name: "users",
		AddColumns: [][]string{
			{"new_field", "TEXT"},
		},
		DropColumns: []string{"old_field"},
	}

	err = s.queries.AlterSQLTable(s.ctx, alterParams)
	assert.NoError(s.T(), err)

	rows, err := s.tx.Query("PRAGMA table_info(users)")
	require.NoError(s.T(), err)
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dfltValue sql.NullString

		err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk)
		require.NoError(s.T(), err)
		columns[name] = true
	}

	assert.True(s.T(), columns["id"])
	assert.True(s.T(), columns["new_field"])
	assert.False(s.T(), columns["old_field"])
}

func (s *QueriesTestSuite) TestDropTable() {
	params := CreateSQLTableParams{
		Name: "temp_table",
		Columns: [][]string{
			{"id", "TEXT", "PRIMARY KEY"},
		},
	}

	err := s.queries.CreateSQLTable(s.ctx, params)
	require.NoError(s.T(), err)

	err = s.queries.DropTable(s.ctx, "temp_table")
	assert.NoError(s.T(), err)

	var count int
	err = s.tx.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='temp_table'").Scan(&count)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, count)
}

func (s *QueriesTestSuite) TestDropTableNotFound() {
	err := s.queries.DropTable(s.ctx, "non_existent_table")
	assert.NoError(s.T(), err)
}

func (s *QueriesTestSuite) TestDropTableIdempotent() {
	params := CreateSQLTableParams{
		Name: "temp_table",
		Columns: [][]string{
			{"id", "TEXT", "PRIMARY KEY"},
		},
	}

	err := s.queries.CreateSQLTable(s.ctx, params)
	require.NoError(s.T(), err)

	err = s.queries.DropTable(s.ctx, "temp_table")
	require.NoError(s.T(), err)

	err = s.queries.DropTable(s.ctx, "temp_table")
	assert.NoError(s.T(), err)
}
