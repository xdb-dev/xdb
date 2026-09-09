---
title: Config
description: Config loading, validation, and defaults for the XDB CLI and daemon.
package: cmd/xdb/cli
---

# Config

XDB reads a JSON config file at `~/.xdb/config.json`. The config controls the daemon, the store backend, and logging. The `cli` package loads the config into typed structs, applies defaults, and validates it.

## Config File

`xdb init` creates the config file and the config directory (`dir`), then starts the daemon. `xdb daemon start` also creates the config file if it is missing. No other command writes the config. The data directory (`<dir>/data`) is created later, when the daemon opens the store:

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

The field order matches `cli.Config`:

```go
type Config struct {
    Dir      string       `json:"dir"`
    Daemon   DaemonConfig `json:"daemon"`
    LogLevel string       `json:"log_level"`
    Store    StoreConfig  `json:"store,omitzero"`
}

type DaemonConfig struct {
    Socket string `json:"socket"`
}

type StoreConfig struct {
    Backend string       `json:"backend,omitempty"`
    FS      FSConfig     `json:"fs,omitzero"`
    Redis   RedisConfig  `json:"redis,omitzero"`
    SQLite  SQLiteConfig `json:"sqlite,omitzero"`
}

type FSConfig struct {
    Dir string `json:"dir,omitempty"`
}

type RedisConfig struct {
    Addr     string `json:"addr,omitempty"`
    Password string `json:"password,omitempty"`
    DB       int    `json:"db,omitempty"`
}

type SQLiteConfig struct {
    Path        string `json:"path,omitempty"`
    Journal     string `json:"journal,omitempty"`
    Sync        string `json:"sync,omitempty"`
    CacheSize   int    `json:"cache_size,omitempty"`
    BusyTimeout int    `json:"busy_timeout,omitempty"`
}
```

## Fields

| Field                      | Default             | Description                                                           |
| -------------------------- | ------------------- | --------------------------------------------------------------------- |
| `dir`                      | `~/.xdb`            | Root directory for all XDB data (must be absolute, or start with `~`) |
| `daemon.socket`            | `xdb.sock`          | Unix socket filename (a bare filename, not a path)                    |
| `store.backend`            | `sqlite`            | Backend: `sqlite`, `memory`, `redis`, or `fs`                         |
| `store.sqlite.path`        | `<datadir>/xdb.db`  | SQLite database file path                                             |
| `store.sqlite.journal`     | `wal`               | Journal mode: `wal`, `delete`, `truncate`, `persist`, `memory`, `off` |
| `store.sqlite.sync`        | `normal`            | Synchronous mode: `off`, `normal`, `full`, `extra`                    |
| `store.sqlite.cache_size`  | `-2000`             | Page cache size in KiB (negative) or pages (positive)                 |
| `store.sqlite.busy_timeout`| `5000`              | Busy timeout in milliseconds                                          |
| `store.redis.addr`         | *(required)*        | Redis server address (`host:port`)                                    |
| `store.redis.password`     | *(empty)*           | Redis auth password                                                   |
| `store.redis.db`           | `0`                 | Redis database number                                                 |
| `store.fs.dir`             | `<datadir>`         | Root directory of the filesystem backend                              |
| `log_level`                | `info`              | Log level: `debug`, `info`, `warn`, or `error`                        |

## Derived Paths

All paths derive from `dir`:

| Helper         | Path                    |
| -------------- | ----------------------- |
| `SocketPath()` | `<dir>/<daemon.socket>` |
| `LogFile()`    | `<dir>/xdb.log`         |
| `PIDFile()`    | `<dir>/<socket-name>.pid` |
| `DataDir()`    | `<dir>/data`            |

## Loading

```go
// Load from an explicit path. The file must exist.
cfg, err := cli.LoadConfig("/path/to/config.json")

// Load from the default path (~/.xdb/config.json). If the file is
// missing, LoadConfig returns validated in-memory defaults and
// writes nothing.
cfg, err := cli.LoadConfig("")

// Create the file with defaults if it is missing. created is true
// when a new file was written.
created, err := cli.EnsureConfigAt("/path/to/config.json")
```

`LoadConfig` unmarshals into `NewDefaultConfig()`, so omitted fields get defaults. It then calls `Validate()` before it returns. `LoadConfig` never writes a file. `EnsureConfigAt` is the only creation path. `xdb init` and `xdb daemon start` call it.

## Validation

`Validate()` returns an error if one of these conditions is false:

- `dir` is non-empty and absolute, or starts with `~`

- `daemon.socket` contains no `/` or `\`

- `log_level` is `debug`, `info`, `warn`, or `error`

- `store.backend` is `sqlite`, `memory`, `fs`, or `redis`

- `store.redis.addr` is set when the backend is `redis`

`SocketPath()` joins `daemon.socket` onto `dir`, so the socket must be a bare filename.

## CLI Flag

The root `--config` / `-c` flag selects the config file:

```
xdb --config /etc/xdb/config.json daemon start
```

The flag defaults to `~/.xdb/config.json`. If you explicitly pass `--config`, a missing file is an error when loading config. Without the flag, a missing default file yields in-memory defaults. `init` and `daemon start` create a missing config before loading it.

## Related Concepts

- [Stores](stores.md): The backend that `store.backend` selects

- [Daemon](daemon.md): Uses the config for the socket path, the log file, and the PID file
