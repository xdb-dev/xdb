package sqlite

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/xdb-dev/xdb"
	"github.com/xdb-dev/xdb/schema"
	"zombiezen.com/go/sqlite"
)

type SQLiteStoreSuite struct {
	suite.Suite
	db     *sqlite.Conn
	schema *schema.Schema
	store  *SQLiteStore
}

func TestSQLiteStoreSuite(t *testing.T) {
	suite.Run(t, &SQLiteStoreSuite{})
}

func (s *SQLiteStoreSuite) SetupSuite() {
	s.schema = &schema.Schema{
		Records: []schema.Record{
			{
				Kind:  "User",
				Table: "users",
				Attributes: []schema.Attribute{
					{Name: "name", Type: schema.String},
					{Name: "age", Type: schema.Int},
					{Name: "is_active", Type: schema.Bool},
					{Name: "created_at", Type: schema.Time},
					{Name: "score", Type: schema.Float},
					{Name: "tags", Type: schema.String, Repeated: true},
				},
			},
		},
	}

	db, err := sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	require.NoError(s.T(), err)

	s.db = db
	s.store = NewSQLiteStore(db, s.schema)

	m := NewMigration(db, s.schema)

	err = m.Run(context.Background())
	require.NoError(s.T(), err)
}

func (s *SQLiteStoreSuite) TearDownSuite() {
	s.db.Close()
}

func (s *SQLiteStoreSuite) TestCRUD() {
	user := createFakeUser()
	ctx := context.Background()

	err := s.store.PutRecord(ctx, user)
	s.Require().NoError(err)

	got, err := s.store.GetRecord(ctx, user.Key())
	s.Require().NoError(err)
	s.EqualValues(user.Key(), got.Key())

	err = s.store.DeleteRecord(ctx, user.Key())
	s.Require().NoError(err)
}

func (s *SQLiteStoreSuite) TestQuery() {
	users := s.createFakeUsers(10)
	ctx := context.Background()

	for _, user := range users {
		err := s.store.PutRecord(ctx, user)
		s.Require().NoError(err)
	}

	q := xdb.Select("*").
		From("User").
		Where("is_active").Eq(true).
		OrderBy("created_at", xdb.DESC).
		Limit(5)

	rs, err := s.store.QueryRecords(ctx, q)
	s.Require().NoError(err)
	s.Len(rs.List(), 5)
}

func (s *SQLiteStoreSuite) createFakeUsers(n int) []*xdb.Record {
	users := make([]*xdb.Record, n)

	for i := 0; i < n; i++ {
		users[i] = createFakeUser()
	}

	return users
}

func createFakeUser() *xdb.Record {
	key := xdb.NewKey("User", gofakeit.UUID())

	return xdb.NewRecord(key,
		xdb.NewTuple(key, "name", gofakeit.Name()),
		xdb.NewTuple(key, "age", gofakeit.IntRange(18, 65)),
		xdb.NewTuple(key, "is_active", gofakeit.Bool()),
		xdb.NewTuple(key, "created_at", gofakeit.Date()),
		xdb.NewTuple(key, "score", gofakeit.Float64()),
		xdb.NewTuple(key, "tags", []string{
			gofakeit.Word(),
			gofakeit.Word(),
			gofakeit.Word(),
		}),
	)
}
