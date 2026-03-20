package sqlgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/filter"
	"github.com/xdb-dev/xdb/schema"
)

var testDef = &schema.Def{
	Fields: map[string]schema.FieldDef{
		"name":   {Type: core.TIDString},
		"status": {Type: core.TIDString},
		"age":    {Type: core.TIDInteger},
		"score":  {Type: core.TIDFloat},
		"active": {Type: core.TIDBoolean},
	},
}

func TestGenerate_Column(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantSQL    string
		wantParams []any
	}{
		{
			name:       "equality",
			expr:       `status == "active"`,
			wantSQL:    `(status = ?)`,
			wantParams: []any{"active"},
		},
		{
			name:       "not equal",
			expr:       `status != "closed"`,
			wantSQL:    `(status != ?)`,
			wantParams: []any{"closed"},
		},
		{
			name:       "greater than",
			expr:       `age > 30`,
			wantSQL:    `(age > ?)`,
			wantParams: []any{int64(30)},
		},
		{
			name:       "greater or equal",
			expr:       `age >= 18`,
			wantSQL:    `(age >= ?)`,
			wantParams: []any{int64(18)},
		},
		{
			name:       "less than",
			expr:       `age < 65`,
			wantSQL:    `(age < ?)`,
			wantParams: []any{int64(65)},
		},
		{
			name:       "less or equal",
			expr:       `score <= 100.0`,
			wantSQL:    `(score <= ?)`,
			wantParams: []any{float64(100.0)},
		},
		{
			name:       "compound AND",
			expr:       `status == "active" && age > 30`,
			wantSQL:    `((status = ?) AND (age > ?))`,
			wantParams: []any{"active", int64(30)},
		},
		{
			name:       "compound OR",
			expr:       `status == "active" || status == "pending"`,
			wantSQL:    `((status = ?) OR (status = ?))`,
			wantParams: []any{"active", "pending"},
		},
		{
			name:       "NOT",
			expr:       `!(active == true)`,
			wantSQL:    `(NOT (active = ?))`,
			wantParams: []any{true},
		},
		{
			name:       "contains",
			expr:       `name.contains("oh")`,
			wantSQL:    `(name LIKE '%' || ? || '%')`,
			wantParams: []any{"oh"},
		},
		{
			name:       "startsWith",
			expr:       `name.startsWith("J")`,
			wantSQL:    `(name LIKE ? || '%')`,
			wantParams: []any{"J"},
		},
		{
			name:       "endsWith",
			expr:       `name.endsWith("hn")`,
			wantSQL:    `(name LIKE '%' || ?)`,
			wantParams: []any{"hn"},
		},
		{
			name:       "size",
			expr:       `size(name) > 3`,
			wantSQL:    `(LENGTH(name) > ?)`,
			wantParams: []any{int64(3)},
		},
		{
			name:       "in list",
			expr:       `status in ["active", "pending"]`,
			wantSQL:    `(status IN (?, ?))`,
			wantParams: []any{"active", "pending"},
		},
		{
			name:       "complex compound with parens",
			expr:       `(status == "active" || status == "pending") && age >= 18`,
			wantSQL:    `(((status = ?) OR (status = ?)) AND (age >= ?))`,
			wantParams: []any{"active", "pending", int64(18)},
		},
		{
			name:       "boolean literal",
			expr:       `active == true`,
			wantSQL:    `(active = ?)`,
			wantParams: []any{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := filter.Compile(tt.expr, testDef)
			require.NoError(t, err)

			wc, err := Generate(f, ColumnStrategy, "")
			require.NoError(t, err)
			assert.Equal(t, tt.wantSQL, wc.SQL)
			assert.Equal(t, tt.wantParams, wc.Params)
		})
	}
}

func TestGenerate_KV(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantSQL    string
		wantParams []any
	}{
		{
			name:       "equality",
			expr:       `status == "active"`,
			wantSQL:    `(_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS TEXT) = ?))`,
			wantParams: []any{"status", "active"},
		},
		{
			name:       "boolean equality",
			expr:       `active == true`,
			wantSQL:    `(_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS REAL) = ?))`,
			wantParams: []any{"active", true},
		},
		{
			name:       "boolean in compound",
			expr:       `active == true && age > 30`,
			wantSQL:    `((_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS REAL) = ?)) AND (_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS REAL) > ?)))`,
			wantParams: []any{"active", true, "age", int64(30)},
		},
		{
			name:       "compound AND",
			expr:       `status == "active" && age > 30`,
			wantSQL:    `((_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS TEXT) = ?)) AND (_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS REAL) > ?)))`,
			wantParams: []any{"status", "active", "age", int64(30)},
		},
		{
			name:       "compound OR",
			expr:       `status == "active" || status == "pending"`,
			wantSQL:    `((_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS TEXT) = ?)) OR (_id IN (SELECT _id FROM posts WHERE _attr = ? AND CAST(_val AS TEXT) = ?)))`,
			wantParams: []any{"status", "active", "status", "pending"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := filter.Compile(tt.expr, testDef)
			require.NoError(t, err)

			wc, err := Generate(f, KVStrategy, "posts")
			require.NoError(t, err)
			assert.Equal(t, tt.wantSQL, wc.SQL)
			assert.Equal(t, tt.wantParams, wc.Params)
		})
	}
}

func TestGenerate_Unsupported(t *testing.T) {
	f, err := filter.Compile(`name.matches("^J.*")`, testDef)
	require.NoError(t, err)

	_, err = Generate(f, ColumnStrategy, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
