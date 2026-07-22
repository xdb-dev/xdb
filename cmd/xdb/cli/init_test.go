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
	// The config created at the custom path still has the default "~/.xdb"
	// dir (init only controls where the config FILE lives, not its
	// contents), so redirect HOME to a temp dir — otherwise the mkdir/spawn
	// steps that follow would touch the developer's real ~/.xdb.
	t.Setenv("HOME", t.TempDir())

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

// TestInitAction_CreatedThenAlreadyExists verifies init's stderr messaging is
// truthful across repeated runs: "Created ..." the first time a config file
// is written, "already exists" on every run after. This is the behavior that
// [LoadConfig] no longer auto-creating the config (W6) makes reliable — init
// is now the sole creator of its own config, so [EnsureConfigAt]'s created
// flag reflects reality instead of racing a prior implicit creation.
func TestInitAction_CreatedThenAlreadyExists(t *testing.T) {
	// The auto-created config's Dir defaults to "~/.xdb"; redirect HOME so
	// the best-effort daemon-spawn step never touches the real home.
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	_, stderr, code := runCLI(t, "--config", configPath, "init")
	require.Equal(t, ExitOK, code, "stderr: %s", stderr)
	assert.Contains(t, stderr, "Created "+configPath)

	_, stderr, code = runCLI(t, "--config", configPath, "init")
	require.Equal(t, ExitOK, code, "stderr: %s", stderr)
	assert.Contains(t, stderr, "already exists")
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
