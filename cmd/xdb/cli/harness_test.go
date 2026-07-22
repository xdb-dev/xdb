package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
	"github.com/xdb-dev/xdb/store"
	"github.com/xdb-dev/xdb/store/xdbmemory"
)

// runCLI runs the CLI in-process with fresh output buffers and returns what
// it wrote to stdout/stderr along with the shell exit code it would produce.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	app := NewAppWithIO(&outBuf, &errBuf)

	err := app.Run(context.Background(), append([]string{"xdb"}, args...))
	FinalizeError(app, err)

	return outBuf.String(), errBuf.String(), ExitCodeFor(err)
}

// tempCLIConfig writes a config JSON pointing at a fresh, isolated temp
// directory with the memory backend, and returns its path plus the
// directory. No daemon is started — the caller decides whether to listen on
// the resulting socket. /tmp is used directly (not t.TempDir()) because Unix
// socket paths have a length limit that t.TempDir()'s nested paths can
// exceed on macOS.
func tempCLIConfig(t *testing.T) (configPath, dir string) {
	t.Helper()

	dir, err := os.MkdirTemp("/tmp", "xdb-cli-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	cfg := map[string]any{
		"dir":    dir,
		"daemon": map[string]any{"socket": "test.sock"},
		"store":  map[string]any{"backend": "memory"},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	configPath = filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(configPath, data, 0o600))

	return configPath, dir
}

// startCLITestDaemon starts a daemon with an in-memory store on a temp
// socket and returns the path to a config file (--config) that points at it.
func startCLITestDaemon(t *testing.T) (configPath string) {
	t.Helper()

	configPath, dir := tempCLIConfig(t)

	sock := filepath.Join(dir, "test.sock")
	s := store.New(xdbmemory.NewDriver())
	router, bus := daemon.NewRouter(s, "test")
	t.Cleanup(bus.Close)

	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)

	srv := &http.Server{Handler: router}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	return configPath
}

func TestRunCLI_CapturesStdout(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "describe", "--errors", "-o", "json")

	assert.Equal(t, ExitOK, code)
	assert.Empty(t, stderr)

	var parsed any
	require.NoError(t, json.Unmarshal([]byte(stdout), &parsed), "stdout must be valid JSON: %s", stdout)
}

func TestRunCLI_CapturesStderrAndExitCode(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "records", "get", "--uri", "xdb://a/b/c")

	assert.Equal(t, ExitConnection, code)
	assert.Contains(t, stderr, "CONNECTION_REFUSED")
	assert.Empty(t, stdout)
}

func TestRunCLI_LiveDaemon(t *testing.T) {
	configPath := startCLITestDaemon(t)

	_, _, code := runCLI(t, "--config", configPath,
		"schemas", "create",
		"--uri", "xdb://ns/posts",
		"--json", `{"fields":{"title":{"type":"string"}}}`,
	)
	require.Equal(t, ExitOK, code)

	_, _, code = runCLI(t, "--config", configPath,
		"records", "create",
		"--uri", "xdb://ns/posts/p1",
		"--json", `{"title":"hello"}`,
	)
	require.Equal(t, ExitOK, code)

	stdout, stderr, code := runCLI(t, "--config", configPath,
		"records", "get",
		"--uri", "xdb://ns/posts/p1",
		"-o", "json",
	)
	require.Equal(t, ExitOK, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "hello")
}

// runCLIStdin runs the CLI with the given string piped as stdin.
// Not safe for parallel tests (swaps the process-global os.Stdin).
func runCLIStdin(t *testing.T, stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	_, err = w.WriteString(stdin)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})

	return runCLI(t, args...)
}
