package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestCompile(t *testing.T) {
	def := &schema.Def{
		Fields: map[string]schema.FieldDef{
			"name":   {Type: core.TIDString},
			"age":    {Type: core.TIDInteger},
			"score":  {Type: core.TIDFloat},
			"active": {Type: core.TIDBoolean},
		},
	}

	tests := []struct {
		name    string
		expr    string
		def     *schema.Def
		wantErr bool
	}{
		{
			name: "simple equality",
			expr: `name == "John"`,
			def:  def,
		},
		{
			name: "numeric comparison",
			expr: `age > 30`,
			def:  def,
		},
		{
			name: "compound AND",
			expr: `name == "John" && age >= 18`,
			def:  def,
		},
		{
			name: "compound OR",
			expr: `name == "John" || name == "Jane"`,
			def:  def,
		},
		{
			name: "negation",
			expr: `!(age < 18)`,
			def:  def,
		},
		{
			name: "contains function",
			expr: `name.contains("oh")`,
			def:  def,
		},
		{
			name: "startsWith function",
			expr: `name.startsWith("J")`,
			def:  def,
		},
		{
			name: "endsWith function",
			expr: `name.endsWith("hn")`,
			def:  def,
		},
		{
			name: "size function",
			expr: `size(name) > 3`,
			def:  def,
		},
		{
			name: "in list",
			expr: `name in ["John", "Jane"]`,
			def:  def,
		},
		{
			name: "boolean field",
			expr: `active == true`,
			def:  def,
		},
		{
			name: "complex compound",
			expr: `(name == "John" || name == "Jane") && age >= 18`,
			def:  def,
		},
		{
			name: "nil schema flexible mode",
			expr: `name == "John"`,
			def:  nil,
		},
		{
			name: "nil schema compound",
			expr: `status == "active" && count > 5`,
			def:  nil,
		},
		{
			name:    "empty expression",
			expr:    "",
			def:     def,
			wantErr: true,
		},
		{
			name:    "bad syntax",
			expr:    `name ==`,
			def:     def,
			wantErr: true,
		},
		{
			name:    "invalid operator",
			expr:    `name ??? "John"`,
			def:     def,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Compile(tt.expr, tt.def)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, f)
			assert.Equal(t, tt.expr, f.Source())
		})
	}
}

func TestMatch(t *testing.T) {
	record := core.NewRecord("test", "users", "1")
	record.Set("name", "John")
	record.Set("age", int64(30))
	record.Set("score", 95.5)
	record.Set("active", true)
	record.Set("bio", "loves coding and coffee")
	record.Set("status", "active")

	def := &schema.Def{
		Fields: map[string]schema.FieldDef{
			"name":   {Type: core.TIDString},
			"age":    {Type: core.TIDInteger},
			"score":  {Type: core.TIDFloat},
			"active": {Type: core.TIDBoolean},
			"bio":    {Type: core.TIDString},
			"status": {Type: core.TIDString},
		},
	}

	tests := []struct {
		name string
		expr string
		want bool
	}{
		// Equality
		{name: "string eq match", expr: `name == "John"`, want: true},
		{name: "string eq no match", expr: `name == "Jane"`, want: false},
		{name: "string ne match", expr: `name != "Jane"`, want: true},
		{name: "string ne no match", expr: `name != "John"`, want: false},

		// Numeric comparisons
		{name: "int gt match", expr: `age > 25`, want: true},
		{name: "int gt no match", expr: `age > 35`, want: false},
		{name: "int lt match", expr: `age < 35`, want: true},
		{name: "int gte exact", expr: `age >= 30`, want: true},
		{name: "int lte exact", expr: `age <= 30`, want: true},
		{name: "float gt match", expr: `score > 90.0`, want: true},
		{name: "float lt no match", expr: `score < 90.0`, want: false},

		// Boolean
		{name: "bool true match", expr: `active == true`, want: true},
		{name: "bool false no match", expr: `active == false`, want: false},

		// String functions
		{name: "contains match", expr: `bio.contains("coding")`, want: true},
		{name: "contains no match", expr: `bio.contains("dancing")`, want: false},
		{name: "startsWith match", expr: `name.startsWith("Jo")`, want: true},
		{name: "startsWith no match", expr: `name.startsWith("Ja")`, want: false},
		{name: "endsWith match", expr: `name.endsWith("hn")`, want: true},
		{name: "endsWith no match", expr: `name.endsWith("ne")`, want: false},
		{name: "size match", expr: `size(name) == 4`, want: true},
		{name: "size no match", expr: `size(name) > 10`, want: false},

		// Compound
		{name: "AND match", expr: `name == "John" && age >= 18`, want: true},
		{name: "AND no match", expr: `name == "John" && age >= 40`, want: false},
		{name: "OR match first", expr: `name == "John" || name == "Jane"`, want: true},
		{name: "OR match second", expr: `name == "Bob" || age == 30`, want: true},
		{name: "OR no match", expr: `name == "Bob" || age == 99`, want: false},

		// Negation
		{name: "NOT match", expr: `!(age < 18)`, want: true},
		{name: "NOT no match", expr: `!(age > 18)`, want: false},

		// In list
		{name: "in match", expr: `status in ["active", "pending"]`, want: true},
		{name: "in no match", expr: `status in ["closed", "archived"]`, want: false},

		// Complex
		{
			name: "complex compound",
			expr: `(name == "John" || name == "Jane") && age >= 18 && active == true`,
			want: true,
		},

		// Missing attribute
		{name: "missing attr", expr: `email == "a@b.com"`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Compile(tt.expr, def)
			require.NoError(t, err)

			got, err := Match(f, record)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatch_NilSchema(t *testing.T) {
	record := core.NewRecord("test", "users", "1")
	record.Set("name", "John")
	record.Set("age", int64(30))

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{name: "dynamic string eq", expr: `name == "John"`, want: true},
		{name: "dynamic int gt", expr: `age > 25`, want: true},
		{name: "dynamic compound", expr: `name == "John" && age > 25`, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Compile(tt.expr, nil)
			require.NoError(t, err)

			got, err := Match(f, record)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
