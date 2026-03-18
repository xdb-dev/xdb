package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// defaultConfig is the initial configuration written by `xdb init`.
var defaultConfig = map[string]any{
	"dir": "~/.xdb",
	"daemon": map[string]any{
		"socket": "xdb.sock",
	},
	"log": map[string]any{
		"level": "info",
		"file":  "xdb.log",
	},
	"store": map[string]any{
		"backend": "memory",
	},
}

func initCmd() *cli.Command {
	return &cli.Command{
		Name:               "init",
		Usage:              "Initialize XDB (create ~/.xdb/, config, start daemon)",
		Category:           "system",
		CustomHelpTemplate: commandHelpTemplate,
		Action:             initAction,
	}
}

func initAction(_ context.Context, cmd *cli.Command) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	xdbDir := filepath.Join(home, ".xdb")

	// Create directory.
	if mkErr := os.MkdirAll(xdbDir, 0o700); mkErr != nil {
		return fmt.Errorf("create %s: %w", xdbDir, mkErr)
	}

	// Write config if it doesn't exist.
	configPath := filepath.Join(xdbDir, "config.json")

	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		data, jsonErr := json.MarshalIndent(defaultConfig, "", "  ")
		if jsonErr != nil {
			return fmt.Errorf("marshal config: %w", jsonErr)
		}

		if writeErr := os.WriteFile(configPath, append(data, '\n'), 0o600); writeErr != nil {
			return fmt.Errorf("write config: %w", writeErr)
		}

		fmt.Fprintf(os.Stderr, "Created %s\n", configPath)
	} else {
		fmt.Fprintf(os.Stderr, "Config already exists: %s\n", configPath)
	}

	return formatOne(cmd, map[string]string{
		"status": "initialized",
		"dir":    xdbDir,
		"config": configPath,
	})
}
