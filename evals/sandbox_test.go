package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testBinary returns the path to bin/xdb. If the binary is not built, it
// skips the test. `make evals` builds the binary.
func testBinary(t *testing.T) string {
	t.Helper()

	path, err := filepath.Abs("../bin/xdb")
	require.NoError(t, err)

	if _, err := os.Stat(path); err != nil {
		t.Skip("bin/xdb not built")
	}

	return path
}

func TestSandbox(t *testing.T) {
	binary := testBinary(t)

	fixtures := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fixtures, "rows.csv"), []byte("a,b\n1,2\n"), 0o600))

	sb, err := NewSandbox(binary, fixtures)
	require.NoError(t, err)

	closed := false
	t.Cleanup(func() {
		if !closed {
			_ = sb.Close()
		}
	})

	assert.True(t, strings.HasPrefix(sb.Root, "/tmp/xdb-eval-"))
	assert.Equal(t, []string{"rows.csv"}, sb.Fixtures)

	ctx := t.Context()
	require.NoError(t, sb.Init(ctx))

	out, err := sb.Run(ctx, "xdb daemon status -o json")
	require.NoError(t, err)
	assert.Equal(t, 0, out.Exit)
	assert.Contains(t, out.Stdout, `"running"`)
	assert.Contains(t, out.Stdout, sb.Home, "daemon socket lives under the sandbox home")

	out, err = sb.Run(ctx, "cat rows.csv && pwd")
	require.NoError(t, err)
	assert.Equal(t, 0, out.Exit)
	assert.Contains(t, out.Stdout, "1,2")
	assert.Contains(t, out.Stdout, sb.Work)

	out, err = sb.Run(ctx, "xdb records get xdb://t/s/missing -o json")
	require.NoError(t, err)
	assert.Equal(t, 1, out.Exit)
	require.NoError(t, AssertExpect(Expect{Error: map[string]any{"code": "NOT_FOUND"}}, out))

	guide, err := sb.Context(ctx)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(guide, "# XDB CLI Context"))

	stray, err := sb.StrayProcesses(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, stray, "daemon runs while the sandbox is open")

	require.NoError(t, sb.Close())
	closed = true

	_, err = os.Stat(sb.Root)
	assert.True(t, os.IsNotExist(err))

	stray, err = sb.StrayProcesses(ctx)
	require.NoError(t, err)
	assert.Empty(t, stray)
}

func TestNewSandboxWithoutFixtures(t *testing.T) {
	binary := testBinary(t)

	sb, err := NewSandbox(binary, filepath.Join(t.TempDir(), "absent"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sb.Close() })

	assert.Empty(t, sb.Fixtures)
}

func TestNewSandboxMissingBinary(t *testing.T) {
	_, err := NewSandbox("/nonexistent/xdb", "")
	require.Error(t, err)
}
