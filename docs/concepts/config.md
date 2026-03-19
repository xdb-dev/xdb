---
title: Configuration
description: Application configuration loading, validation, and defaults for the XDB CLI and daemon.
package: cmd/xdb/cli
---

# Configuration

XDB uses a JSON configuration file at `~/.xdb/config.json` to control the daemon, store backend, and logging. The `cli` package provides typed config loading with validation and sensible defaults.

## Config File

Created automatically by `xdb init` (which also creates the data directory and starts the daemon) or on first `daemon start`:

```json
{
  "dir": "~/.xdb",
  "daemon": {
    "socket": "xdb.sock"
  },
  "store": {
    "backend": "sqlite"
  },
  "log_level": "info"
}
```

## Structure

```go
type Config struct {
    Dir      string       `json:"dir"`
    Daemon   DaemonConfig `json:"daemon"`
    Store    StoreConfig  `json:"store,omitzero"`
    LogLevel string       `json:"log_level"`
}

type DaemonConfig struct {
    Socket string `json:"socket"`
}

type StoreConfig struct {
    Backend string       `json:"backend,omitempty"`
    SQLite  SQLiteConfig `json:"sqlite,omitzero"`
    Redis   RedisConfig  `json:"redis,omitzero"`
    FS      FSConfig     `json:"fs,omitzero"`
}

type SQLiteConfig struct {
    Path        string `json:"path,omitempty"`
    Journal     string `json:"journal,omitempty"`
    Sync        string `json:"sync,omitempty"`
    CacheSize   int    `json:"cache_size,omitempty"`
    BusyTimeout int    `json:"busy_timeout,omitempty"`
}

type RedisConfig struct {
    Addr     string `json:"addr,omitempty"`
    Password string `json:"password,omitempty"`
    DB       int    `json:"db,omitempty"`
}

type FSConfig struct {
    Dir string `json:"dir,omitempty"`
}
```

## Fields

| Field                      | Default             | Description                                                           |
| -------------------------- | ------------------- | --------------------------------------------------------------------- |
| `dir`                      | `~/.xdb`            | Root directory for all XDB data (must be absolute or start with `~`)  |
| `daemon.socket`            | `xdb.sock`          | Unix socket filename (must be a filename, not a path)                 |
| `store.backend`            | `sqlite`            | Store backend: `sqlite`, `memory`, `redis`, or `fs`                   |
| `store.sqlite.path`        | `<datadir>/xdb.db`  | SQLite database file path                                             |
| `store.sqlite.journal`     | `wal`               | Journal mode: `wal`, `delete`, `truncate`, `persist`, `memory`, `off` |
| `store.sqlite.sync`        | `normal`            | Synchronous mode: `off`, `normal`, `full`, `extra`                    |
| `store.sqlite.cache_size`  | `-2000`             | Page cache size in KiB (negative) or pages (positive)                 |
| `store.sqlite.busy_timeout`| `5000`              | Busy timeout in milliseconds                                          |
| `store.redis.addr`         | *(required)*        | Redis server address (`host:port`)                                    |
| `store.redis.password`     | *(empty)*           | Redis auth password                                                   |
| `store.redis.db`           | `0`                 | Redis database number                                                 |
| `store.fs.dir`             | `<datadir>`         | Filesystem store root directory                                       |
| `log_level`                | `info`              | Log level: `debug`, `info`, `warn`, or `error`                        |

## Derived Paths

All paths are derived from `dir`:

| Helper         | Path             |
| -------------- | ---------------- |
| `SocketPath()` | `<dir>/xdb.sock` |
| `LogFile()`    | `<dir>/xdb.log`  |
| `PIDFile()`    | `<dir>/xdb.pid`  |
| `DataDir()`    | `<dir>/data`     |

## Loading

```go
// Load from explicit path.
cfg, err := cli.LoadConfig("/path/to/config.json")

// Load from default path (~/.xdb/config.json), creating if missing.
cfg, err := cli.LoadConfig("")
```

`LoadConfig` unmarshals into `NewDefaultConfig()`, so omitted fields get defaults. It then calls `Validate()` before returning.

## Validation

`Validate()` checks:

- `dir` is non-empty and absolute (or starts with `~`)
- `daemon.socket` is a filename (no `/` or `\`)
- `log_level` is one of the recognized levels
- `store.backend` is a supported backend name
- `store.redis.addr` is required when backend is `redis`

## CLI Flag

The root `--config` / `-c` flag overrides the default path:

```
xdb --config /etc/xdb/config.json daemon start
```

All subcommands that need config read this flag via `cmd.Root().String("config")`.

## Related Concepts

- [Stores](stores.md) — Storage backends configured via `store.backend`
- [Daemon](daemon.md) — Uses config for socket path, log file, and PID file
