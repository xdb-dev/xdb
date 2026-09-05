package cli

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
)

func TestIsDaemonRunning_NoPIDFile(t *testing.T) {
	dir := t.TempDir()
	cfg := NewDefaultConfig()
	cfg.Dir = dir

	assert.False(t, isDaemonRunning(cfg))
}

func TestIsDaemonRunning_StalePIDFile(t *testing.T) {
	dir := t.TempDir()
	cfg := NewDefaultConfig()
	cfg.Dir = dir

	// Write a PID that doesn't correspond to a running process.
	pidFile := cfg.PIDFile()
	require.NoError(t, os.WriteFile(pidFile, []byte("999999\n"), 0o600))

	assert.False(t, isDaemonRunning(cfg))
}

func TestIsDaemonRunning_CurrentProcess(t *testing.T) {
	dir := t.TempDir()
	cfg := NewDefaultConfig()
	cfg.Dir = dir

	// Write our own PID — we are alive.
	pidFile := cfg.PIDFile()
	require.NoError(t, daemon.WritePID(pidFile))

	assert.True(t, isDaemonRunning(cfg))
}

func TestSpawnDaemon_IdempotentWhenRunning(t *testing.T) {
	dir := t.TempDir()
	cfg := NewDefaultConfig()
	cfg.Dir = dir

	// Simulate a running daemon by writing our own PID.
	pidFile := cfg.PIDFile()
	require.NoError(t, daemon.WritePID(pidFile))

	// spawnDaemon should succeed (no-op) when daemon is already running.
	err := spawnDaemon(cfg, "", io.Discard)
	assert.NoError(t, err, "spawnDaemon should be idempotent when daemon is running")
}
