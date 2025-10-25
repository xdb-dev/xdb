package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gojekfarm/xtools/xload"
	"github.com/gojekfarm/xtools/xload/providers/yaml"
	"github.com/xdb-dev/xdb/driver/xdbmemory"
	"github.com/xdb-dev/xdb/driver/xdbsqlite"
)

type Config struct {
	// Address to bind the HTTP server to.
	// Default: :8080
	Addr string `env:"ADDR"`

	// Directory to store schema files.
	// Default: .schema/
	SchemaDir string `env:"SCHEMA_DIR"`

	// Store configuration.
	// Only one of the following stores can be configured.
	Store struct {
		SQLite *xdbsqlite.Config `env:",prefix=SQLITE_"`
		Memory *xdbmemory.Config `env:",prefix=MEMORY_"`
	} `env:",prefix=STORE_"`
}

func NewDefaultConfig() *Config {
	cfg := &Config{}

	cfg.Addr = ":8080"
	cfg.SchemaDir = ".schema"

	return cfg
}

func LoadConfig(ctx context.Context, configPath string) (*Config, error) {
	cfg := NewDefaultConfig()

	loader, err := createLoader(configPath)
	if err != nil {
		return cfg, err
	}

	err = xload.Load(ctx, cfg, xload.WithLoader(loader))

	return cfg, err
}

// createLoader creates the appropriate loader chain based on config file availability
func createLoader(configPath string) (xload.Loader, error) {
	var yamlPath string

	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		yamlPath = configPath
	} else {
		yamlPath = findDefaultConfigFile()
	}

	if yamlPath != "" {
		slog.Info("[HTTP] Using YAML config file", "path", yamlPath)

		yamlLoader, err := yaml.NewFileLoader(yamlPath, "_")
		if err != nil {
			return nil, fmt.Errorf("failed to create YAML loader: %w", err)
		}

		return xload.SerialLoader(yamlLoader, xload.OSLoader()), nil
	}

	return xload.SerialLoader(xload.OSLoader()), nil
}

// findDefaultConfigFile checks for default config files and returns the first one found
func findDefaultConfigFile() string {
	candidates := []string{"xdb.yaml", "xdb.yml"}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
