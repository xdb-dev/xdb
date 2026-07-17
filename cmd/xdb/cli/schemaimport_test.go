package cli

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/api"
	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
	"github.com/xdb-dev/xdb/rpc/client"
	"github.com/xdb-dev/xdb/schema"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

const fixtures = "testdata/schemaimport"

func fixture(name string) string {
	return filepath.Join(fixtures, name)
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		data    string
		want    srcFormat
		wantErr bool
	}{
		{"proto by ext", "user.proto", "", formatProto, false},
		{"json by ext", "user.json", "", formatJSONSchema, false},
		{"schema.json by ext", "user.schema.json", "", formatJSONSchema, false},
		{"json by content", "user.txt", "{\"type\":\"object\"}", formatJSONSchema, false},
		{"proto by content", "user.txt", "syntax = \"proto3\";", formatProto, false},
		{"unknown", "user.txt", "hello", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := detectFormat(tc.path, []byte(tc.data))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoadSchemas_Proto(t *testing.T) {
	defs, format, err := loadSchemas(context.Background(), fixture("user.proto"), "com.example", nil)
	require.NoError(t, err)
	assert.Equal(t, formatProto, format)
	require.Len(t, defs, 1)

	def := defs[0]
	assert.Equal(t, "xdb://com.example/User", def.URI.String())
	assert.Contains(t, def.Fields, "name")
	assert.Contains(t, def.Fields, "email")
	assert.Equal(t, "2", def.Fields["email"].Annotations["proto.number"])
}

func TestLoadSchemas_JSONSchema(t *testing.T) {
	defs, format, err := loadSchemas(context.Background(), fixture("user.schema.json"), "com.example", nil)
	require.NoError(t, err)
	assert.Equal(t, formatJSONSchema, format)
	require.Len(t, defs, 1)

	def := defs[0]
	assert.Equal(t, "xdb://com.example/user", def.URI.String())
	assert.True(t, def.Fields["name"].Required)
	assert.False(t, def.Fields["email"].Required)
}

func TestComputeDelta_NewSchema(t *testing.T) {
	imported := mustLoad(t, "user.proto", "com.example")

	d := computeDelta(nil, imported, formatProto)
	assert.True(t, d.NewSchema)
	assert.True(t, d.hasDrift())
	assert.Len(t, d.Added, 2)
}

func TestComputeDelta_InSync(t *testing.T) {
	def := mustLoad(t, "user.proto", "com.example")

	d := computeDelta(def, def, formatProto)
	assert.False(t, d.hasDrift())
	assert.Empty(t, d.Added)
	assert.Empty(t, d.Removed)
	assert.Empty(t, d.Renames)
}

func TestComputeDelta_AddedField(t *testing.T) {
	stored := mustLoad(t, "user.proto", "com.example")
	imported := mustLoad(t, "user_v2.proto", "com.example")

	d := computeDelta(stored, imported, formatProto)
	assert.True(t, d.hasDrift())
	require.Len(t, d.Added, 1)
	assert.Equal(t, "age", d.Added[0].Name)
	assert.Empty(t, d.Renames)
}

func TestComputeDelta_ProtoRename(t *testing.T) {
	stored := mustLoad(t, "user.proto", "com.example")
	imported := mustLoad(t, "user_rename.proto", "com.example")

	d := computeDelta(stored, imported, formatProto)
	require.Len(t, d.Renames, 1)
	assert.Equal(t, "email", d.Renames[0].From)
	assert.Equal(t, "contact_email", d.Renames[0].To)
	assert.True(t, d.Renames[0].Proto)
	// The renamed field is not double-counted as a plain add/remove.
	assert.Empty(t, d.Added)
	assert.Empty(t, d.Removed)
}

func TestComputeDelta_HeuristicRename(t *testing.T) {
	stored := mustLoad(t, "user.schema.json", "com.example")
	imported := mustLoad(t, "user_rename.schema.json", "com.example")

	d := computeDelta(stored, imported, formatJSONSchema)
	require.Len(t, d.Renames, 1)
	assert.Equal(t, "email", d.Renames[0].From)
	assert.Equal(t, "contact", d.Renames[0].To)
	assert.False(t, d.Renames[0].Proto)
}

func TestUnresolvedRenames(t *testing.T) {
	proto := renamePair{From: "a", To: "b", Proto: true}
	heur := renamePair{From: "email", To: "contact"}

	// Proto renames never block.
	assert.Empty(t, unresolvedRenames([]renamePair{proto}, nil, false))

	// A non-proto rename blocks without acknowledgment.
	assert.Len(t, unresolvedRenames([]renamePair{heur}, nil, false), 1)

	// --yes clears all.
	assert.Empty(t, unresolvedRenames([]renamePair{heur}, nil, true))

	// --rename email:contact clears the matching pair.
	assert.Empty(t, unresolvedRenames([]renamePair{heur}, map[string]string{"email": "contact"}, false))

	// A mismatched --rename does not clear it.
	assert.Len(t, unresolvedRenames([]renamePair{heur}, map[string]string{"email": "other"}, false), 1)
}

func TestParseRenameFlags(t *testing.T) {
	m, err := parseRenameFlags([]string{"a:b", "c:d"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "b", "c": "d"}, m)

	_, err = parseRenameFlags([]string{"bad"})
	require.Error(t, err)
}

// --- Integration against a live daemon ---

func startSchemaDaemon(t *testing.T) *client.Client {
	t.Helper()

	dir, err := os.MkdirTemp("/tmp", "xdb-schemaimport-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	sock := filepath.Join(dir, "test.sock")
	router := daemon.NewRouter(xdbmemory.New(), "test")

	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)

	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return client.New(sock)
}

func TestSchemaImportDiff_E2E(t *testing.T) {
	a := &App{client: startSchemaDaemon(t)}
	ctx := context.Background()
	var out bytes.Buffer

	// Import creates the schema.
	require.NoError(t, a.doImport(ctx, importParams{
		path: fixture("user.proto"),
		ns:   "com.example",
	}, &out))

	var got api.GetSchemaResponse
	require.NoError(t, a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{
		URI: "xdb://com.example/User",
	}, &got))
	require.NotNil(t, got.Data)
	assert.Contains(t, got.Data.Fields, "email")

	// Diff against the same file: no drift.
	drift, err := a.doDiff(ctx, importParams{path: fixture("user.proto"), ns: "com.example"}, &out)
	require.NoError(t, err)
	assert.False(t, drift)

	// Editing the source (add a field) makes diff report drift.
	drift, err = a.doDiff(ctx, importParams{path: fixture("user_v2.proto"), ns: "com.example"}, &out)
	require.NoError(t, err)
	assert.True(t, drift)

	// Re-import applies the edit.
	require.NoError(t, a.doImport(ctx, importParams{path: fixture("user_v2.proto"), ns: "com.example"}, &out))

	// Diff passes again.
	drift, err = a.doDiff(ctx, importParams{path: fixture("user_v2.proto"), ns: "com.example"}, &out)
	require.NoError(t, err)
	assert.False(t, drift)
}

func TestSchemaImport_NonProtoRenameRefused(t *testing.T) {
	a := &App{client: startSchemaDaemon(t)}
	ctx := context.Background()
	var out bytes.Buffer

	// Seed the schema with name + email.
	require.NoError(t, a.doImport(ctx, importParams{
		path: fixture("user.schema.json"),
		ns:   "com.acme",
	}, &out))

	// A JSON-Schema rename (email -> contact) is refused without --rename/--yes.
	err := a.doImport(ctx, importParams{
		path: fixture("user_rename.schema.json"),
		ns:   "com.acme",
	}, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to import")

	// Acknowledging with --rename lets it through.
	require.NoError(t, a.doImport(ctx, importParams{
		path:    fixture("user_rename.schema.json"),
		ns:      "com.acme",
		renames: map[string]string{"email": "contact"},
	}, &out))

	var got api.GetSchemaResponse
	require.NoError(t, a.client.Call(ctx, "schemas.get", &api.GetSchemaRequest{
		URI: "xdb://com.acme/user",
	}, &got))
	assert.Contains(t, got.Data.Fields, "contact")
}

func TestDataSchemaDescription(t *testing.T) {
	def := mustLoad(t, "user.proto", "com.example")
	def.Description = "a user"

	doc := dataSchemaDescription(def.URI.String(), def)
	assert.Equal(t, "a user", doc["description"])
	assert.Equal(t, "strict", doc["mode"])
	assert.Contains(t, doc, "revision")
	assert.Contains(t, doc, "annotations")

	fields, ok := doc["fields"].([]map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, fields)
}

func mustLoad(t *testing.T, name, ns string) *schema.Def {
	t.Helper()
	defs, _, err := loadSchemas(context.Background(), fixture(name), ns, nil)
	require.NoError(t, err)
	require.Len(t, defs, 1)

	return defs[0]
}
