---
title: Daemon
description: Background daemon lifecycle management including spawning, stopping, and status checking.
package: cmd/xdb/cli, cmd/xdb/daemon
---

# Daemon

The XDB daemon runs a JSON-RPC server over a Unix socket, providing access to all store operations. The CLI manages the daemon lifecycle through `xdb daemon start|stop|status|restart`. The daemon is also started automatically by `xdb init`.

## Architecture

The daemon uses a **parent-child spawn pattern**:

```
xdb daemon start
├── Parent process (CLI)
│   ├── Loads config
│   ├── Checks for existing daemon (PID file)
│   ├── Re-execs itself with XDB_DAEMON_CHILD=1
│   ├── Waits for socket to accept connections
│   └── Exits with success message
│
└── Child process (daemon)
    ├── Detached from parent (setsid)
    ├── Stdout/stderr redirected to log file
    ├── Writes PID file
    ├── Starts JSON-RPC server on Unix socket
    └── Runs until SIGTERM/SIGINT
```

The child process outlives the parent CLI command. It runs in its own session (`setsid`) so it isn't affected by terminal closure.

## Commands

### Start

```
xdb daemon start              # Background (default)
xdb daemon start --foreground # Blocks in current process
```

Background mode:

1. Reads config via `--config` flag or default `~/.xdb/config.json`
2. Checks PID file — returns successfully if daemon is already running, cleans stale PIDs
3. Opens log file for append
4. Re-execs the binary with `XDB_DAEMON_CHILD=1` environment variable
5. Waits up to 3 seconds for the socket to accept connections
6. Prints PID and socket path, then exits

All commands are idempotent: `start` is a no-op when already running, `stop` is a no-op when already stopped.

Foreground mode (or when `XDB_DAEMON_CHILD=1` is set):

1. Registers signal handler for `SIGINT` and `SIGTERM`
2. Calls `daemon.Start(ctx)` which blocks until shutdown

### Stop

```
xdb daemon stop
```

1. Reads PID from PID file
2. Sends `SIGTERM` to the process
3. Polls for up to 5 seconds for the process to exit
4. Cleans up PID file

### Status

```
xdb daemon status
```

Reports `running` or `stopped`, socket path, and PID (when running).

### Restart

```
xdb daemon restart
```

Stops the daemon (if running), then starts it.

## Files

| File              | Purpose                     |
| ----------------- | --------------------------- |
| `~/.xdb/xdb.sock` | Unix socket for JSON-RPC    |
| `~/.xdb/xdb.pid`  | PID of the running daemon   |
| `~/.xdb/xdb.log`  | Daemon stdout/stderr output |

## Daemon Package

The `cmd/xdb/daemon` package contains the server implementation:

- `daemon.Config` — `SocketPath`, `LogFile`, `Version`
- `daemon.New(cfg)` — creates a `Daemon` instance
- `daemon.Start(ctx)` — starts the HTTP server on the Unix socket (blocks)
- `daemon.Stop()` — graceful shutdown with 5-second timeout
- `daemon.NewRouter(store, version)` — creates the JSON-RPC router with all services registered

The CLI layer (`cmd/xdb/cli/daemon.go`) handles the process lifecycle (spawn, signal, PID management) on top of the daemon package.

## Related Concepts

- [Configuration](config.md) — Config file that controls daemon behavior
- [Stores](stores.md) — Storage backend used by the daemon
