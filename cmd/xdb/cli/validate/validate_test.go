package validate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURI(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{
			name: "valid URI",
			raw:  "xdb://ns/schema/id",
		},
		{
			name:    "control characters",
			raw:     "xdb://ns/schema/\x01id",
			wantErr: "control characters",
		},
		{
			name:    "query params",
			raw:     "xdb://ns/schema/id?foo=bar",
			wantErr: "query parameters",
		},
		{
			name:    "path traversal",
			raw:     "xdb://ns/../schema/id",
			wantErr: "path traversal",
		},
		{
			name:    "percent encoding",
			raw:     "xdb://ns/schema/id%20name",
			wantErr: "percent-encoding",
		},
		{
			name:    "space",
			raw:     "xdb://ns/schema/ id",
			wantErr: "whitespace",
		},
		{
			name:    "tab",
			raw:     "xdb://ns/schema/x\tid",
			wantErr: "whitespace",
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: "", // ParseURI should reject this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URI(tt.raw)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else if tt.name == "valid URI" {
				assert.NoError(t, err)
			}
			// For empty string, we just check it returns some error
			if tt.raw == "" {
				assert.Error(t, err)
			}
		})
	}
}

func TestFilePath(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Create a temp file inside CWD for testing
	tmpFile := filepath.Join(cwd, "testfile_validate_tmp")
	err = os.WriteFile(tmpFile, []byte("test"), 0o644)
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile) })

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name: "valid relative path",
			path: "testfile_validate_tmp",
		},
		{
			name: "valid absolute path in CWD",
			path: tmpFile,
		},
		{
			name:    "path outside CWD",
			path:    "/tmp/somefile",
			wantErr: "outside working directory",
		},
		{
			name:    "dangerous prefix /dev/",
			path:    "/dev/null",
			wantErr: "dangerous path",
		},
		{
			name:    "dangerous prefix /proc/",
			path:    "/proc/self/status",
			wantErr: "dangerous path",
		},
		{
			name:    "dangerous prefix /etc/",
			path:    "/etc/passwd",
			wantErr: "dangerous path",
		},
		{
			name:    "dangerous prefix /sys/",
			path:    "/sys/kernel",
			wantErr: "dangerous path",
		},
		{
			name: "nonexistent file in CWD",
			path: "nonexistent_file_abc123.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FilePath(tt.path)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.True(t, filepath.IsAbs(result), "result should be absolute path")
				assert.True(
					t,
					strings.HasPrefix(result, cwd),
					"result %q should be under CWD %q", result, cwd,
				)
			}
		})
	}
}

func TestPayload(t *testing.T) {
	validJSON := []byte(`{"key": "value"}`)
	largeJSON := make([]byte, 0, 200)
	largeJSON = append(largeJSON, []byte(`{"k":"`)...)
	largeJSON = append(largeJSON, make([]byte, 190)...)
	largeJSON = append(largeJSON, []byte(`"}`)...)

	// Build deeply nested JSON (21 levels)
	deepJSON := buildNestedJSON(21)

	// Build exactly 20 levels (should pass)
	okNestJSON := buildNestedJSON(20)

	tests := []struct {
		name    string
		data    []byte
		maxSize int
		wantErr string
	}{
		{
			name:    "valid payload",
			data:    validJSON,
			maxSize: 1024,
		},
		{
			name:    "oversized payload",
			data:    validJSON,
			maxSize: 5,
			wantErr: "exceeds maximum size",
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid`),
			maxSize: 1024,
			wantErr: "invalid JSON",
		},
		{
			name:    "deeply nested JSON",
			data:    deepJSON,
			maxSize: 1024 * 1024,
			wantErr: "nesting depth",
		},
		{
			name:    "nesting at limit",
			data:    okNestJSON,
			maxSize: 1024 * 1024,
		},
		{
			name:    "nested with arrays",
			data:    []byte(`{"a":[[[[[[[[[[[[[[[[[[[[[1]]]]]]]]]]]]]]]]]]]]]}`),
			maxSize: 1024 * 1024,
			wantErr: "nesting depth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Payload(tt.data, tt.maxSize)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func buildNestedJSON(depth int) []byte {
	// Build {"a":{"a":...}} to the given depth
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString(`{"a":`)
	}
	b.WriteString(`1`)
	for i := 0; i < depth; i++ {
		b.WriteString(`}`)
	}
	data := []byte(b.String())
	// Sanity: must be valid JSON
	if !json.Valid(data) {
		panic("buildNestedJSON produced invalid JSON")
	}
	return data
}

func TestMutuallyExclusive(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]bool
		wantErr string
	}{
		{
			name:  "no flags set",
			flags: map[string]bool{"json": false, "yaml": false},
		},
		{
			name:  "one flag set",
			flags: map[string]bool{"json": true, "yaml": false},
		},
		{
			name:    "two flags set",
			flags:   map[string]bool{"json": true, "yaml": true},
			wantErr: "specify only one of --json, --yaml",
		},
		{
			name:    "three flags set",
			flags:   map[string]bool{"alpha": true, "beta": true, "gamma": true},
			wantErr: "specify only one of --alpha, --beta, --gamma",
		},
		{
			name:  "empty map",
			flags: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MutuallyExclusive(tt.flags)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
