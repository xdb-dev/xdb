package xdb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	q := Select("name", "age").
		From("users").
		Where("age").Eq(30).
		And("name").Ne("Alice").
		And("name").In("Bob", "Charlie").
		And("birthdate").Lt(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)).
		And("birthdate").Gt(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)).
		And("name").NotIn("David", "Eve").
		OrderBy("name", "asc", "age", "desc").
		Skip(10).Limit(20)

	expected := `SELECT name, age FROM users WHERE age = '30' AND name != 'Alice' AND name IN ('Bob', 'Charlie') AND name NOT IN ('David', 'Eve') AND birthdate < '2000-01-01 00:00:00 +0000 UTC' AND birthdate > '1970-01-01 00:00:00 +0000 UTC' ORDER BY name asc, age desc OFFSET 10 LIMIT 20`

	got := q.String()
	assert.Equal(t, expected, got)
}
