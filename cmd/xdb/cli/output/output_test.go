package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/cmd/xdb/cli/output"
)

type testItem struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

func TestJSONFormatter(t *testing.T) {
	f := output.New(output.FormatJSON)

	t.Run("FormatOne", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatOne(&buf, testItem{Name: "alice", Count: 3})
		require.NoError(t, err)
		assert.Equal(t, "{\n  \"name\": \"alice\",\n  \"count\": 3\n}\n", buf.String())
	})

	t.Run("FormatList", func(t *testing.T) {
		var buf bytes.Buffer
		items := []any{
			testItem{Name: "alice", Count: 3},
			testItem{Name: "bob", Count: 7},
		}
		err := f.FormatList(&buf, items)
		require.NoError(t, err)
		expected := "[\n  {\n    \"name\": \"alice\",\n    \"count\": 3\n  },\n  {\n    \"name\": \"bob\",\n    \"count\": 7\n  }\n]\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatError", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatError(&buf, errors.New("something broke"))
		require.NoError(t, err)
		assert.Equal(t, "{\n  \"error\": \"something broke\"\n}\n", buf.String())
	})

	t.Run("FormatList empty", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatList(&buf, nil)
		require.NoError(t, err)
		assert.Equal(t, "[]\n", buf.String())
	})
}

func TestNDJSONFormatter(t *testing.T) {
	f := output.New(output.FormatNDJSON)

	t.Run("FormatOne", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatOne(&buf, testItem{Name: "alice", Count: 3})
		require.NoError(t, err)
		assert.Equal(t, "{\"name\":\"alice\",\"count\":3}\n", buf.String())
	})

	t.Run("FormatList", func(t *testing.T) {
		var buf bytes.Buffer
		items := []any{
			testItem{Name: "alice", Count: 3},
			testItem{Name: "bob", Count: 7},
		}
		err := f.FormatList(&buf, items)
		require.NoError(t, err)
		expected := "{\"name\":\"alice\",\"count\":3}\n{\"name\":\"bob\",\"count\":7}\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatError", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatError(&buf, errors.New("something broke"))
		require.NoError(t, err)
		assert.Equal(t, "{\"error\":\"something broke\"}\n", buf.String())
	})

	t.Run("FormatList empty", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatList(&buf, nil)
		require.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})
}

func TestTableFormatter(t *testing.T) {
	f := output.New(output.FormatTable)

	t.Run("FormatOne struct", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatOne(&buf, testItem{Name: "alice", Count: 3})
		require.NoError(t, err)
		expected := "name    alice\ncount   3\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatOne map", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatOne(&buf, map[string]any{"name": "alice", "count": 3})
		require.NoError(t, err)
		expected := "count   3\nname    alice\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatList structs", func(t *testing.T) {
		var buf bytes.Buffer
		items := []any{
			testItem{Name: "alice", Count: 3},
			testItem{Name: "bob", Count: 7},
		}
		err := f.FormatList(&buf, items)
		require.NoError(t, err)
		expected := "name    count\nalice   3\nbob     7\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatList maps", func(t *testing.T) {
		var buf bytes.Buffer
		items := []any{
			map[string]any{"name": "alice", "count": 3},
			map[string]any{"name": "bob", "count": 7},
		}
		err := f.FormatList(&buf, items)
		require.NoError(t, err)
		expected := "count   name\n3       alice\n7       bob\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatError", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatError(&buf, errors.New("something broke"))
		require.NoError(t, err)
		assert.Equal(t, "Error: something broke\n", buf.String())
	})

	t.Run("FormatList empty", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatList(&buf, nil)
		require.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})
}

func TestYAMLFormatter(t *testing.T) {
	f := output.New(output.FormatYAML)

	t.Run("FormatOne", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatOne(&buf, testItem{Name: "alice", Count: 3})
		require.NoError(t, err)
		assert.Equal(t, "---\nname: alice\ncount: 3\n", buf.String())
	})

	t.Run("FormatList", func(t *testing.T) {
		var buf bytes.Buffer
		items := []any{
			testItem{Name: "alice", Count: 3},
			testItem{Name: "bob", Count: 7},
		}
		err := f.FormatList(&buf, items)
		require.NoError(t, err)
		expected := "---\n- name: alice\n  count: 3\n- name: bob\n  count: 7\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("FormatError", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatError(&buf, errors.New("something broke"))
		require.NoError(t, err)
		assert.Equal(t, "---\nerror: something broke\n", buf.String())
	})

	t.Run("FormatList empty", func(t *testing.T) {
		var buf bytes.Buffer
		err := f.FormatList(&buf, nil)
		require.NoError(t, err)
		assert.Equal(t, "---\n[]\n", buf.String())
	})
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		isTTY   bool
		expect  output.Format
		wantErr bool
	}{
		{
			name:    "unknown format",
			flag:    "xml",
			isTTY:   false,
			wantErr: true,
		},
		{
			name:    "wrong case",
			flag:    "JSON",
			isTTY:   false,
			wantErr: true,
		},
		{
			name:   "explicit ndjson",
			flag:   "ndjson",
			isTTY:  true,
			expect: output.FormatNDJSON,
		},
		{
			name:   "explicit table",
			flag:   "table",
			isTTY:  false,
			expect: output.FormatTable,
		},
		{
			name:   "explicit json",
			flag:   "json",
			isTTY:  true,
			expect: output.FormatJSON,
		},
		{
			name:   "explicit yaml",
			flag:   "yaml",
			isTTY:  false,
			expect: output.FormatYAML,
		},
		{
			name:   "tty default is table",
			flag:   "",
			isTTY:  true,
			expect: output.FormatTable,
		},
		{
			name:   "non-tty default is json",
			flag:   "",
			isTTY:  false,
			expect: output.FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := output.Detect(tt.flag, tt.isTTY)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestFormatPage(t *testing.T) {
	page := output.Page{
		Items:      []any{map[string]string{"_id": "a"}, map[string]string{"_id": "b"}},
		Total:      5,
		NextOffset: 2,
	}

	t.Run("json renders envelope", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatJSON).FormatPage(&buf, page))

		var doc map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
		assert.Equal(t, float64(5), doc["total"])
		assert.Equal(t, float64(2), doc["next_offset"])
		assert.Len(t, doc["items"], 2)
	})

	t.Run("json omits next_offset when zero", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatJSON).FormatPage(&buf, output.Page{Items: []any{}, Total: 0}))
		assert.NotContains(t, buf.String(), "next_offset")
		assert.Contains(t, buf.String(), `"items"`)
	})

	t.Run("ndjson streams bare items", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatNDJSON).FormatPage(&buf, page))

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, lines, 2)
		assert.NotContains(t, buf.String(), "total")
	})

	t.Run("yaml renders envelope", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatYAML).FormatPage(&buf, page))
		assert.Contains(t, buf.String(), "total: 5")
	})

	t.Run("table renders items only", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, output.New(output.FormatTable).FormatPage(&buf, page))
		assert.Contains(t, buf.String(), "a")
		assert.NotContains(t, buf.String(), "next_offset")
	})
}

func TestJSONNoHTMLEscaping(t *testing.T) {
	var buf bytes.Buffer
	env := &output.ErrorEnvelope{Code: "X", Message: "m", Hint: "run xdb describe <resource>.<action>"}
	require.NoError(t, output.New(output.FormatJSON).FormatError(&buf, env))
	assert.Contains(t, buf.String(), "<resource>")
	assert.NotContains(t, buf.String(), "u003c")
}
