package evals

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want XDBCall
		ok   bool
	}{
		{
			name: "not xdb",
			cmd:  "ls -la && cat foo.csv",
			ok:   false,
		},
		{
			name: "uri only is not a call",
			cmd:  "echo xdb://ns/s/id",
			ok:   false,
		},
		{
			name: "records create",
			cmd:  `xdb records create xdb://ns/s/id --json '{"a":1}' -o json`,
			want: XDBCall{Resource: "records", Action: "create", Layer: LayerNone},
			ok:   true,
		},
		{
			name: "after a pipe",
			cmd:  `cat rows.ndjson | xdb import --uri xdb://ns/s -f - | tee out`,
			want: XDBCall{Resource: "import", Layer: LayerNone},
			ok:   true,
		},
		{
			name: "after cd and &&",
			cmd:  "cd /tmp/work && xdb schemas get xdb://ns/s",
			want: XDBCall{Resource: "schemas", Action: "get", Layer: LayerNone},
			ok:   true,
		},
		{
			name: "quoted xdb token in payload is not a segment",
			cmd:  `xdb records list xdb://ns/s --filter 'title == "xdb"'`,
			want: XDBCall{Resource: "records", Action: "list", Layer: LayerNone},
			ok:   true,
		},
		{
			name: "root alias",
			cmd:  "xdb ls xdb://ns/s",
			want: XDBCall{Resource: "records", Action: "list", Layer: LayerNone},
			ok:   true,
		},
		{
			name: "context",
			cmd:  "xdb context",
			want: XDBCall{Resource: "context", Discovery: true, Layer: LayerContext},
			ok:   true,
		},
		{
			name: "root help",
			cmd:  "xdb --help",
			want: XDBCall{Discovery: true, Layer: LayerOverview},
			ok:   true,
		},
		{
			name: "bare xdb is root help",
			cmd:  "xdb",
			want: XDBCall{Discovery: true, Layer: LayerOverview},
			ok:   true,
		},
		{
			name: "resource help",
			cmd:  "xdb records -h",
			want: XDBCall{Resource: "records", Discovery: true, Layer: LayerOverview},
			ok:   true,
		},
		{
			name: "describe actions",
			cmd:  "xdb describe --actions",
			want: XDBCall{Resource: "describe", Discovery: true, Layer: LayerOverview},
			ok:   true,
		},
		{
			name: "action help",
			cmd:  "xdb records create --help",
			want: XDBCall{Resource: "records", Action: "create", Discovery: true, Layer: LayerAction},
			ok:   true,
		},
		{
			name: "describe action",
			cmd:  "xdb describe records.create",
			want: XDBCall{Resource: "records", Action: "create", Discovery: true, Layer: LayerAction},
			ok:   true,
		},
		{
			name: "describe filter maps to records list",
			cmd:  "xdb describe --filter",
			want: XDBCall{Resource: "records", Action: "list", Discovery: true, Layer: LayerReference},
			ok:   true,
		},
		{
			name: "describe uri",
			cmd:  "xdb describe --uri xdb://ns/s -o json",
			want: XDBCall{Resource: "describe", Discovery: true, Layer: LayerReference},
			ok:   true,
		},
		{
			name: "describe errors",
			cmd:  "xdb describe --errors",
			want: XDBCall{Resource: "describe", Discovery: true, Layer: LayerReference},
			ok:   true,
		},
		{
			name: "describe type",
			cmd:  "xdb describe Record",
			want: XDBCall{Resource: "describe", Discovery: true, Layer: LayerReference},
			ok:   true,
		},
		{
			name: "skills list",
			cmd:  "xdb skills",
			want: XDBCall{Resource: "skills", Discovery: true, Layer: LayerSkills},
			ok:   true,
		},
		{
			name: "skills get",
			cmd:  "xdb skills get bulk-data",
			want: XDBCall{Resource: "skills", Action: "bulk-data", Discovery: true, Layer: LayerSkills},
			ok:   true,
		},
		{
			name: "for loop",
			cmd:  "for id in a b; do xdb records delete xdb://ns/s/$id --force; done",
			want: XDBCall{Resource: "records", Action: "delete", Layer: LayerNone},
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyCommand(tt.cmd)
			require.Equal(t, tt.ok, ok)

			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestContextExamples(t *testing.T) {
	guide := "```bash\n" +
		"# Read\n" +
		"xdb records get  xdb://ns/s/id --fields title\n" +
		"xdb records list xdb://ns/s --filter 'x' -o ndjson\n" +
		"xdb schemas create xdb://ns/s --json '{}'\n" +
		"xdb describe --actions\n" +
		"```\n" +
		"| Action | `xdb describe <resource>.<action>` |\n"

	got := ContextExamples(guide)

	assert.True(t, got["records get"])
	assert.True(t, got["records list"])
	assert.True(t, got["schemas create"])
	assert.False(t, got["describe --actions"])
	assert.False(t, got["records delete"])
}
