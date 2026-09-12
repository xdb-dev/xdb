package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
	"github.com/xdb-dev/xdb/core"
	"github.com/xdb-dev/xdb/schema"
)

func TestYAMLFormatter_JSONShape(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		expect string
	}{
		{
			name: "schema definition uses its JSON form",
			value: &schema.Def{
				URI:      core.MustParseURI("xdb://com.example/posts"),
				Mode:     schema.ModeStrict,
				Revision: 3,
				Fields: map[string]schema.Field{
					"title": {Type: core.TypeString, Required: true},
					"tags":  {Type: core.NewArrayType(core.TIDString)},
					"lines": {
						Type: core.NewArrayType(core.TIDJSON),
						Items: map[string]schema.Field{
							"sku": {Type: core.TypeString, Required: true},
						},
					},
				},
			},
			expect: "---\n" +
				"fields:\n" +
				"    lines:\n" +
				"        items:\n" +
				"            sku:\n" +
				"                type: string\n" +
				"                required: true\n" +
				"        type: array\n" +
				"        elem_type: json\n" +
				"    tags:\n" +
				"        type: array\n" +
				"        elem_type: string\n" +
				"    title:\n" +
				"        type: string\n" +
				"        required: true\n" +
				"uri: xdb://com.example/posts\n" +
				"mode: strict\n" +
				"revision: 3\n",
		},
		{
			name: "raw JSON is a nested document",
			value: map[string]any{
				"dry_run": true,
				"valid":   true,
				"would":   "create",
				"record":  json.RawMessage(`{"_id":"p1","title":"hi","views":42}`),
			},
			expect: "---\n" +
				"dry_run: true\n" +
				"record:\n" +
				"    _id: p1\n" +
				"    title: hi\n" +
				"    views: 42\n" +
				"valid: true\n" +
				"would: create\n",
		},
		{
			name: "strings that read as other types stay quoted",
			value: map[string]string{
				"at":   "2026-09-11T12:04:06Z",
				"flag": "true",
				"id":   "007",
			},
			expect: "---\n" +
				"at: \"2026-09-11T12:04:06Z\"\n" +
				"flag: \"true\"\n" +
				"id: \"007\"\n",
		},
	}

	f := output.New(output.FormatYAML)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, f.FormatOne(&buf, tt.value))
			assert.Equal(t, tt.expect, buf.String())
		})
	}
}

func TestYAMLFormatter_SchemaDefPage(t *testing.T) {
	def := &schema.Def{
		URI:  core.MustParseURI("xdb://com.example/events"),
		Mode: schema.ModeFlexible,
	}

	var buf bytes.Buffer
	err := output.New(output.FormatYAML).FormatPage(&buf, output.Page{
		Items: []any{def},
		Total: 1,
	})
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "uri: xdb://com.example/events")
	assert.Contains(t, buf.String(), "mode: flexible")
	assert.Contains(t, buf.String(), "total: 1")
}
