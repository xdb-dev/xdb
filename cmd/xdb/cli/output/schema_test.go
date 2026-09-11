package output_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestTableFormatter_SchemaDef(t *testing.T) {
	tests := []struct {
		name   string
		def    *schema.Def
		expect string
	}{
		{
			name: "fields with markers, system fields last",
			def: schema.StampSystemFields(&schema.Def{
				URI:      core.MustParseURI("xdb://com.example/posts"),
				Mode:     schema.ModeStrict,
				Revision: 3,
				Fields: map[string]schema.Field{
					"title":  {Type: core.TypeString, Required: true},
					"views":  {Type: core.TypeInt},
					"author": {Type: core.TypeString, Indexed: true},
					"slug":   {Type: core.TypeString, Indexed: true, Unique: true},
					"tags":   {Type: core.NewArrayType(core.TIDString)},
					"lines": {
						Type: core.NewArrayType(core.TIDJSON),
						Items: map[string]schema.Field{
							"sku": {Type: core.TypeString, Required: true},
							"qty": {Type: core.TypeInt},
						},
					},
				},
			}),
			expect: "" +
				"uri        xdb://com.example/posts\n" +
				"mode       strict\n" +
				"revision   3\n" +
				"fields     author     string (indexed)\n" +
				"           lines      array<json>\n" +
				"             qty      integer\n" +
				"             sku      string (required)\n" +
				"           slug       string (indexed, unique)\n" +
				"           tags       array<string>\n" +
				"           title      string (required)\n" +
				"           views      integer\n" +
				"           _updated   time\n" +
				"           _version   integer\n",
		},
		{
			name: "description and no fields",
			def: &schema.Def{
				URI:         core.MustParseURI("xdb://com.example/events"),
				Description: "Audit events",
				Mode:        schema.ModeFlexible,
				Revision:    1,
			},
			expect: "" +
				"uri           xdb://com.example/events\n" +
				"description   Audit events\n" +
				"mode          flexible\n" +
				"revision      1\n",
		},
	}

	f := output.New(output.FormatTable)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, f.FormatOne(&buf, tt.def))
			assert.Equal(t, tt.expect, buf.String())
		})
	}
}

func TestTableFormatter_SchemaDefPage(t *testing.T) {
	events := &schema.Def{
		URI:      core.MustParseURI("xdb://com.example/events"),
		Mode:     schema.ModeFlexible,
		Revision: 1,
		Fields: map[string]schema.Field{
			"kind": {Type: core.TypeString},
			"at":   {Type: core.TypeTime},
		},
	}
	posts := &schema.Def{
		URI:      core.MustParseURI("xdb://com.example/posts"),
		Mode:     schema.ModeStrict,
		Revision: 3,
		Fields: map[string]schema.Field{
			"title":    {Type: core.TypeString},
			"views":    {Type: core.TypeInt},
			"_version": {Type: core.TypeInt},
			"_updated": {Type: core.TypeTime},
		},
	}

	var buf bytes.Buffer
	err := output.New(output.FormatTable).FormatPage(&buf, output.Page{
		Items: []any{events, posts},
		Total: 2,
	})
	require.NoError(t, err)

	expect := "" +
		"uri                        mode       revision   fields\n" +
		"xdb://com.example/events   flexible   1          2\n" +
		"xdb://com.example/posts    strict     3          4\n"
	assert.Equal(t, expect, buf.String())
}
