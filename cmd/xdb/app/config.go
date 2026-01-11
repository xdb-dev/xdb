package app

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultConfigDir    = "~/.xdb"
	defaultConfigFile   = "config.json"
	defaultDaemonAddr   = "localhost:8147"
	defaultDaemonSocket = "xdb.sock"
	defaultLogLevel     = "info"
)

// DaemonConfig holds the daemon-specific configuration.
type DaemonConfig struct {
	Addr   string `json:"addr"`
	Socket string `json:"socket"`
}

// Config holds the XDB application configuration.
type Config struct {
	Dir      string       `json:"dir"`
	Daemon   DaemonConfig `json:"daemon"`
	LogLevel string       `json:"log_level"`
}

// PIDFile returns the path to the PID file.
func (c *Config) PIDFile() string {
	return filepath.Join(c.expandedDir(), "xdb.pid")
}

// LogFile returns the path to the log file.
func (c *Config) LogFile() string {
	return filepath.Join(c.expandedDir(), "xdb.log")
}

// SocketPath returns the path to the Unix socket.
func (c *Config) SocketPath() string {
	return filepath.Join(c.expandedDir(), c.Daemon.Socket)
}

// DataDir returns the path to the data directory.
func (c *Config) DataDir() string {
	return filepath.Join(c.expandedDir(), "data")
}

// Addr returns the daemon address for backward compatibility.
func (c *Config) Addr() string {
	return c.Daemon.Addr
}

func (c *Config) expandedDir() string {
	return expandTilde(c.Dir)
}

// NewDefaultConfig creates a new Config with default values.
func NewDefaultConfig() *Config {
	return &Config{
		Dir: defaultConfigDir,
		Daemon: DaemonConfig{
			Addr:   defaultDaemonAddr,
			Socket: defaultDaemonSocket,
		},
		LogLevel: defaultLogLevel,
	}
}

// ConfigPath returns the default config file path.
func ConfigPath() string {
	return filepath.Join(expandTilde(defaultConfigDir), defaultConfigFile)
}

// EnsureConfig checks if the config file exists and creates it with defaults if not.
// Returns true if a new config was created.
func EnsureConfig() (bool, error) {
	configPath := ConfigPath()
	configDir := filepath.Dir(configPath)

	if _, err := os.Stat(configPath); err == nil {
		return false, nil
	}

	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return false, fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg := NewDefaultConfig()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		return false, fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created config: %s\n", configPath)
	return true, nil
}

// LoadConfig loads the configuration from the JSON file.
// If configPath is empty, it uses the default path (~/.xdb/config.json).
// It will create the config with defaults if it doesn't exist.
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		if _, err := EnsureConfig(); err != nil {
			return nil, err
		}
		configPath = ConfigPath()
	}

	data, err := os.ReadFile(configPath) // #nosec G304 - configPath is from trusted CLI flag or hardcoded default
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := NewDefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate checks that the config values are valid.
func (c *Config) Validate() error {
	configPath := ConfigPath()

	if c.Dir == "" {
		return fmt.Errorf("dir cannot be empty\nEdit %s to fix", configPath)
	}

	expandedDir := expandTilde(c.Dir)
	if !filepath.IsAbs(expandedDir) && !strings.HasPrefix(c.Dir, "~") {
		return fmt.Errorf("dir must be an absolute path or start with ~\nEdit %s to fix", configPath)
	}

	if c.Daemon.Addr == "" {
		return fmt.Errorf("daemon.addr cannot be empty\nEdit %s to fix", configPath)
	}

	if _, _, err := net.SplitHostPort(c.Daemon.Addr); err != nil {
		return fmt.Errorf("daemon.addr must be a valid host:port format (e.g., localhost:8147)\nEdit %s to fix", configPath)
	}

	if c.Daemon.Socket == "" {
		return fmt.Errorf("daemon.socket cannot be empty\nEdit %s to fix", configPath)
	}

	if strings.ContainsAny(c.Daemon.Socket, "/\\") {
		return fmt.Errorf("daemon.socket must be a filename, not a path\nEdit %s to fix", configPath)
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log_level must be one of: debug, info, warn, error\nEdit %s to fix", configPath)
	}

	return nil
}

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if path == "~" {
		return home
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}

	return path
}
