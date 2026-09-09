package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDaemonStart_PrintsRealPID checks that daemon start prints a positive
// child PID. The PID must be captured before Process.Release invalidates it.
// The test builds and runs the xdb binary in a temporary directory.
func TestDaemonStart_PrintsRealPID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary build + daemon spawn in -short mode")
	}

	xdbDir := t.TempDir()
	bin := filepath.Join(xdbDir, "xdb")

	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "github.com/xdb-dev/xdb/cmd/xdb")
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "failed to build xdb binary")

	// Fresh config + data dir isolated from any developer daemon.
	runDir := t.TempDir()
	cfg := map[string]any{
		"dir":       runDir,
		"log_level": "info",
		"daemon":    map[string]any{"addr": "localhost:0", "socket": "xdb.sock"},
		"store":     map[string]any{"backend": "memory"},
	}

	cfgPath := filepath.Join(runDir, "config.json")
	cfgBytes, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgPath, cfgBytes, 0o600))

	// Ensure we stop the daemon even if assertions fail.
	t.Cleanup(func() {
		stop := exec.CommandContext(t.Context(), bin, "--config", cfgPath, "daemon", "stop")
		stop.Run()
	})

	var stderr bytes.Buffer

	start := exec.CommandContext(t.Context(), bin, "--config", cfgPath, "daemon", "start")
	start.Stderr = &stderr
	start.Stdout = &stderr
	start.Env = append(os.Environ(), "HOME="+runDir)

	done := make(chan error, 1)
	go func() { done <- start.Run() }()

	select {
	case err := <-done:
		require.NoError(t, err, "daemon start failed: %s", stderr.String())
	case <-time.After(10 * time.Second):
		_ = start.Process.Kill()
		t.Fatalf("daemon start timed out; output so far:\n%s", stderr.String())
	}

	out := stderr.String()
	t.Logf("daemon start output:\n%s", out)

	re := regexp.MustCompile(`PID (-?\d+)`)
	matches := re.FindStringSubmatch(out)
	require.Len(t, matches, 2, "expected 'PID <n>' in output, got:\n%s", out)

	pid, convErr := strconv.Atoi(matches[1])
	require.NoError(t, convErr)
	assert.Positive(t, pid, "daemon PID must be > 0 (regression: Release() zeroed child.Process.Pid)")
}
