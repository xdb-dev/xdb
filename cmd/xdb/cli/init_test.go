package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// writeTestConfig writes a config file with dir pointing to a temp subdirectory
// that does not yet exist. Returns the config path and the xdb dir path.
func writeTestConfig(t *testing.T) (configPath, xdbDir string) {
	t.Helper()

	dir := t.TempDir()
	configPath = filepath.Join(dir, "config.json")
	xdbDir = filepath.Join(dir, "xdb-data")

	cfg := NewDefaultConfig()
	cfg.Dir = xdbDir

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, data, 0o600))

	return configPath, xdbDir
}

func initApp(action cli.ActionFunc, configPath string) *cli.Command {
	return &cli.Command{
		Name: "xdb",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: configPath,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "init",
				Action: action,
			},
		},
	}
}

func TestInitAction_HonorsConfigFlag(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom", "config.json")

	app := &cli.Command{
		Name: "xdb",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "~/.xdb/config.json",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "init",
				Action: initAction,
			},
		},
	}

	err := app.Run(context.Background(), []string{"xdb", "--config", configPath, "init"})
	require.NoError(t, err)

	// Config file should be created at the custom path, not the default.
	_, statErr := os.Stat(configPath)
	assert.NoError(t, statErr, "config file should exist at custom path")
}

func TestInitAction_CreatesXDBDir(t *testing.T) {
	configPath, xdbDir := writeTestConfig(t)
	app := initApp(initAction, configPath)

	err := app.Run(context.Background(), []string{"xdb", "init"})
	require.NoError(t, err)

	// Init should create the XDB data directory.
	info, statErr := os.Stat(xdbDir)
	require.NoError(t, statErr, "xdb dir should be created")
	assert.True(t, info.IsDir())
}

func TestInitAction_SucceedsWhenDaemonSpawnFails(t *testing.T) {
	// Init should succeed even when daemon spawn fails (best-effort).
	// In test environment, spawnDaemon will fail because os.Executable()
	// returns the test binary, not the xdb binary.
	configPath, _ := writeTestConfig(t)
	app := initApp(initAction, configPath)

	err := app.Run(context.Background(), []string{"xdb", "init"})
	require.NoError(t, err, "init should succeed even if daemon spawn fails")

	// Config should still exist.
	_, statErr := os.Stat(configPath)
	assert.NoError(t, statErr)
}

func TestInitAction_UsesDefaultWhenNoFlag(t *testing.T) {
	// When --config is not set, it should use DefaultConfigPath().
	// We can't easily test this without writing to ~/.xdb, so just
	// verify the function resolves the flag correctly with the default value.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	app := initApp(initAction, configPath)

	err := app.Run(context.Background(), []string{"xdb", "init"})
	require.NoError(t, err)

	_, statErr := os.Stat(configPath)
	assert.NoError(t, statErr, "config file should exist at default flag path")
}
