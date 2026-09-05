---
title: Daemon
description: Background daemon lifecycle. Spawn, stop, status, and restart.
package: cmd/xdb/cli, cmd/xdb/daemon
---

# Daemon

The XDB daemon runs a JSON-RPC server over a Unix socket. The server exposes all store operations. The CLI manages the daemon lifecycle through `xdb daemon start|stop|status|restart`. `xdb init` also starts the daemon.

## Architecture

The daemon uses a **parent-child spawn pattern**. The CLI re-execs its own binary, so the daemon is a separate process that outlives the CLI command:

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

The child process outlives the parent CLI command. It runs in its own session (`setsid`), so the closure of the terminal does not affect it.

## Commands

### Start

```
xdb daemon start              # Background (default)
xdb daemon start --foreground # Blocks in current process
```

Background mode:

1. Creates the config file with defaults if it is missing (`EnsureConfigAt`)
2. Reads the config from the `--config` flag, or from the default `~/.xdb/config.json`
3. Creates `dir` if it is missing
4. Reads the PID file. If the daemon is already running, returns successfully. Removes a stale PID file
5. Opens the log file for append
6. Re-execs the binary with the `XDB_DAEMON_CHILD=1` environment variable
7. Waits up to 3 seconds for the socket to accept connections
8. Prints the PID and the socket path, then exits

All commands are idempotent. `start` is a no-op when the daemon is already running. `stop` is a no-op when the daemon is already stopped.

Foreground mode (`--foreground`, or when `XDB_DAEMON_CHILD=1` is set):

1. Opens the store that the config selects
2. Registers a signal handler for `SIGINT` and `SIGTERM`
3. Calls `(*Daemon).Start(ctx, store)`, which blocks until shutdown

### Stop

```
xdb daemon stop
```

1. Reads the PID from the PID file
2. Sends `SIGTERM` to the process
3. Polls for up to 5 seconds for the process to exit
4. Removes the PID file

### Status

```
xdb daemon status
```

Reports `running` or `stopped`, the socket path, and the PID (when running). When the daemon is stopped, the exit code is 2. `--quiet` suppresses the output and leaves only the exit code, so a script can gate on `xdb daemon status --quiet && ...`.

### Restart

```
xdb daemon restart
```

Stops the daemon if it is running, then starts it.

## Change Streams

The daemon owns an in-process event bus. The record, schema, and batch services publish a change notification after each successful mutation (after the commit, for batches). `watch` streams deliver the notifications as server-sent events. A stream always begins with a `ready` frame before any event, so a client knows that the subscription is live. Delivery is at-most-once with no replay. Only watchers connected to this daemon at the time of the change see an event.

## Files

| File              | Purpose                     |
| ----------------- | --------------------------- |
| `<dir>/xdb.sock`  | Unix socket for JSON-RPC (`<dir>/<daemon.socket>`) |
| `<dir>/<socket-name>.pid` | PID of the running daemon. The CLI reads this path |
| `<dir>/xdb.log`   | Daemon stdout and stderr    |

`<dir>` is `~/.xdb` by default. The daemon writes its PID file next to the socket, with the socket name and a `.pid` extension (`daemon.PIDPath`). The CLI derives the same path with `Config.PIDFile()`, so you can change `daemon.socket` and `stop` and `status` still find the daemon.

## Daemon Package

The `cmd/xdb/daemon` package contains the server implementation:

- `daemon.Config` — `SocketPath`, `LogFile`, `Version`
- `daemon.New(cfg)` — creates a `Daemon`
- `(*Daemon).Start(ctx, store)` — writes the PID file and serves JSON-RPC on the Unix socket. Blocks until `ctx` is canceled
- `(*Daemon).Stop()` — graceful shutdown with a 5-second timeout
- `daemon.NewRouter(store, version)` — returns `(*rpc.Router, *api.Bus)` with all services registered. The caller owns the bus and must close it on shutdown, so that watch streams end cleanly
- `daemon.PIDPath(socketPath)` — the PID file path for a socket path

The CLI layer (`cmd/xdb/cli/daemon.go`) handles the process lifecycle (spawn, signal, PID management) on top of the daemon package.

## Related Concepts

- [Configuration](config.md) — The config file that controls the daemon
- [Stores](stores.md) — The backend that the daemon opens
