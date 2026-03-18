package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Expr
		wantErr bool
	}{
		{
			name:  "equals",
			input: "name=John",
			want:  &Expr{Attr: "name", Op: OpEq, Value: "John"},
		},
		{
			name:  "not equals",
			input: "name!=John",
			want:  &Expr{Attr: "name", Op: OpNe, Value: "John"},
		},
		{
			name:  "greater than",
			input: "age>30",
			want:  &Expr{Attr: "age", Op: OpGt, Value: "30"},
		},
		{
			name:  "less than",
			input: "age<30",
			want:  &Expr{Attr: "age", Op: OpLt, Value: "30"},
		},
		{
			name:  "greater than or equal",
			input: "age>=30",
			want:  &Expr{Attr: "age", Op: OpGte, Value: "30"},
		},
		{
			name:  "less than or equal",
			input: "age<=30",
			want:  &Expr{Attr: "age", Op: OpLte, Value: "30"},
		},
		{
			name:  "contains",
			input: "name contains oh",
			want:  &Expr{Attr: "name", Op: OpContains, Value: "oh"},
		},
		{
			name:  "quoted value",
			input: `name="hello world"`,
			want:  &Expr{Attr: "name", Op: OpEq, Value: "hello world"},
		},
		{
			name:  "whitespace around operator",
			input: "name = John",
			want:  &Expr{Attr: "name", Op: OpEq, Value: "John"},
		},
		{
			name:  "dotted attribute path",
			input: "author.id=42",
			want:  &Expr{Attr: "author.id", Op: OpEq, Value: "42"},
		},
		{
			name:  "contains with extra whitespace",
			input: "title  contains  hello",
			want:  &Expr{Attr: "title", Op: OpContains, Value: "hello"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "no operator",
			input:   "justanattribute",
			wantErr: true,
		},
		{
			name:    "empty attr",
			input:   "=value",
			wantErr: true,
		},
		{
			name:    "empty value",
			input:   "attr=",
			wantErr: true,
		},
		{
			name:    "empty quoted value",
			input:   `attr=""`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Attr, got.Attr)
			assert.Equal(t, tt.want.Op, got.Op)
			assert.Equal(t, tt.want.Value, got.Value)
		})
	}
}

func TestMatch(t *testing.T) {
	record := core.NewRecord("test", "users", "1")
	record.Set("name", "John")
	record.Set("age", int64(30))
	record.Set("score", 95.5)
	record.Set("bio", "loves coding and coffee")

	tests := []struct {
		name   string
		expr   *Expr
		record *core.Record
		want   bool
	}{
		{
			name:   "string equality match",
			expr:   &Expr{Attr: "name", Op: OpEq, Value: "John"},
			record: record,
			want:   true,
		},
		{
			name:   "string equality no match",
			expr:   &Expr{Attr: "name", Op: OpEq, Value: "Jane"},
			record: record,
			want:   false,
		},
		{
			name:   "not equals match",
			expr:   &Expr{Attr: "name", Op: OpNe, Value: "Jane"},
			record: record,
			want:   true,
		},
		{
			name:   "not equals no match",
			expr:   &Expr{Attr: "name", Op: OpNe, Value: "John"},
			record: record,
			want:   false,
		},
		{
			name:   "numeric greater than match",
			expr:   &Expr{Attr: "age", Op: OpGt, Value: "25"},
			record: record,
			want:   true,
		},
		{
			name:   "numeric greater than no match",
			expr:   &Expr{Attr: "age", Op: OpGt, Value: "35"},
			record: record,
			want:   false,
		},
		{
			name:   "numeric less than match",
			expr:   &Expr{Attr: "age", Op: OpLt, Value: "35"},
			record: record,
			want:   true,
		},
		{
			name:   "numeric greater than or equal match exact",
			expr:   &Expr{Attr: "age", Op: OpGte, Value: "30"},
			record: record,
			want:   true,
		},
		{
			name:   "numeric less than or equal match exact",
			expr:   &Expr{Attr: "age", Op: OpLte, Value: "30"},
			record: record,
			want:   true,
		},
		{
			name:   "float numeric comparison",
			expr:   &Expr{Attr: "score", Op: OpGt, Value: "90.0"},
			record: record,
			want:   true,
		},
		{
			name:   "contains match",
			expr:   &Expr{Attr: "bio", Op: OpContains, Value: "coding"},
			record: record,
			want:   true,
		},
		{
			name:   "contains no match",
			expr:   &Expr{Attr: "bio", Op: OpContains, Value: "dancing"},
			record: record,
			want:   false,
		},
		{
			name:   "missing attribute",
			expr:   &Expr{Attr: "email", Op: OpEq, Value: "a@b.com"},
			record: record,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.expr, tt.record)
			assert.Equal(t, tt.want, got)
		})
	}
}
