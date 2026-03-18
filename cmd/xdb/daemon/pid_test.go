package daemon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdb-dev/xdb/cmd/xdb/daemon"
)

func TestWritePID_ReadPID_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pid")

	err := daemon.WritePID(path)
	require.NoError(t, err)

	pid, err := daemon.ReadPID(path)
	require.NoError(t, err)
	assert.Equal(t, os.Getpid(), pid)
}

func TestReadPID_NonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.pid")

	_, err := daemon.ReadPID(path)
	require.Error(t, err)
}

func TestRemovePID_ExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pid")

	err := os.WriteFile(path, []byte("12345\n"), 0o600)
	require.NoError(t, err)

	err = daemon.RemovePID(path)
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestRemovePID_NonExistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.pid")

	err := daemon.RemovePID(path)
	require.NoError(t, err)
}
