# CLI Command Implementation Patterns

## Two Patterns

The app package uses two patterns depending on command complexity:

### 1. Direct Handler Pattern (get, put, list, remove, make-schema)

Simple commands use a single file with a handler function that delegates to `App`:

```go
func Get(ctx context.Context, cmd *cli.Command) error {
    // 1. Parse and validate arguments
    uriStr := cmd.StringArg("uri")
    if uriStr == "" {
        return ErrURIRequired
    }

    // 2. Load config and initialize app
    cfg, err := LoadConfig(cmd.String("config"))
    if err != nil {
        return err
    }

    app, err := New(cfg)
    if err != nil {
        return err
    }
    defer app.Shutdown(ctx)

    // 3. Call business logic
    data, err := app.GetByURI(ctx, uri)
    if err != nil {
        return err
    }

    // 4. Format and write output
    format := getOutputFormat(cmd)
    writer := NewOutputWriter(os.Stdout, format)
    return writer.Write(data)
}
```

### 2. Struct Pattern (daemon)

Complex command groups use two files:

- `daemon.go` - CLI action handlers (thin wrappers)
- `daemon_control.go` - Business logic in struct methods

```go
// daemon_control.go
type Daemon struct {
    Config *Config
}

func NewDaemon(cfg *Config) *Daemon {
    return &Daemon{Config: cfg}
}

func (d *Daemon) Start() error { }
func (d *Daemon) Stop(force bool) error { }

// daemon.go
func DaemonStart(ctx context.Context, cmd *cli.Command) error {
    cfg, err := LoadConfig(cmd.String("config"))
    if err != nil {
        return err
    }

    daemon := NewDaemon(cfg)
    return daemon.Start()
}
```

## Key Types

- **App** (`app.go`) - Core application struct holding config and store drivers. Created via `New(cfg)`, used by get/put/list/remove/make-schema handlers.
- **Daemon** (`daemon_control.go`) - Manages daemon lifecycle (start/stop/status/restart). Created via `NewDaemon(cfg)`.
- **Config** (`config.go`) - Configuration with store backend settings (memory, sqlite, redis, fs).
- **storeSet** (`store.go`) - Groups store interfaces (SchemaStore, TupleStore, RecordStore, HealthChecker) with cleanup functions. Initialized by `initStoreFromConfig()`.

## File Organization

| File | Purpose |
|------|---------|
| `app.go` | App struct and business logic methods |
| `config.go` | Config loading, validation, defaults |
| `daemon.go` | Daemon CLI handlers |
| `daemon_control.go` | Daemon lifecycle implementation |
| `errors.go` | Sentinel errors |
| `get.go`, `put.go`, `list.go`, `remove.go` | CRUD CLI handlers |
| `make_schema.go` | Schema creation handler |
| `output.go` | Formatters (JSON, Table, YAML) and OutputWriter |
| `store.go` | Store backend initialization (memory, sqlite, redis, fs) |
| `server.go` | Internal HTTP server (used by daemon) |

## Anti-Patterns

### Mixing CLI and Business Logic

```go
// Bad: CLI dependency in business logic
func (d *Daemon) Stop(cmd *cli.Command) error {
    force := cmd.Bool("force")
}

// Good: Handler extracts parameters, passes values
func DaemonStop(ctx context.Context, cmd *cli.Command) error {
    force := cmd.Bool("force")
    daemon := NewDaemon(cfg)
    return daemon.Stop(force)
}
```
