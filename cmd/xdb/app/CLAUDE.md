# CLI Command Implementation Pattern

Commands use a struct-based pattern for better encapsulation and testability.

## Structure

Each command group (e.g., daemon) uses two files:

1. `<command>.go` - CLI action handlers
2. `<command>_control.go` - Implementation with struct methods

## Example

### Implementation File (`daemon_control.go`)

```go
type Daemon struct {
    Config *Config
}

func NewDaemon(cfg *Config) *Daemon {
    return &Daemon{Config: cfg}
}

// Public methods - exported API
func (d *Daemon) Start() error { }
func (d *Daemon) Stop(force bool) error { }
func (d *Daemon) GetStatus(ctx context.Context) (*DaemonStatusInfo, error) { }

// Private methods - internal helpers
func (d *Daemon) readPID() (int, error) { }
func (d *Daemon) writePID(pid int) error { }

// Package functions - stateless utilities
func IsProcessRunning(pid int) bool { }
func PrintDaemonStatus(info *DaemonStatusInfo, asJSON bool) error { }
```

### Command File (`daemon.go`)

```go
func DaemonStart(ctx context.Context, cmd *cli.Command) error {
    cfg, err := LoadConfig("")
    if err != nil {
        return err
    }

    daemon := NewDaemon(cfg)
    return daemon.Start()
}

func DaemonStop(ctx context.Context, cmd *cli.Command) error {
    cfg, err := LoadConfig("")
    if err != nil {
        return err
    }

    force := cmd.Bool("force")
    daemon := NewDaemon(cfg)
    return daemon.Stop(force)
}
```

## Organization Rules

**Public Methods** - Exported API, may be called from other packages

**Private Methods** - Internal helpers, access shared state (Config)

**Package Functions** - Stateless utilities, pure presentation/computation logic

## Pattern: CLI Handler

All CLI handlers follow this pattern:

```go
func CommandAction(ctx context.Context, cmd *cli.Command) error {
    // 1. Load config
    cfg, err := LoadConfig("")
    if err != nil {
        return err
    }

    // 2. Extract CLI parameters
    force := cmd.Bool("force")

    // 3. Create struct and call method
    daemon := NewDaemon(cfg)
    return daemon.Method(force)
}
```

## Anti-Patterns

### ❌ Bad: Passing Config Everywhere

```go
func StartDaemon(cfg *Config) error { }
func StopDaemon(cfg *Config, force bool) error { }
```

### ✅ Good: Encapsulate Config in Struct

```go
type Daemon struct { Config *Config }
func (d *Daemon) Start() error { }
func (d *Daemon) Stop(force bool) error { }
```

### ❌ Bad: Exposing Internal Helpers

```go
func ReadPID(cfg *Config) (int, error) { }
```

### ✅ Good: Private Methods

```go
func (d *Daemon) readPID() (int, error) { }
```

### ❌ Bad: Mixing CLI and Business Logic

```go
func (d *Daemon) Stop(cmd *cli.Command) error {
    force := cmd.Bool("force")  // CLI dependency in business logic
}
```

### ✅ Good: Separate Concerns

```go
// CLI handler extracts parameters
func DaemonStop(ctx context.Context, cmd *cli.Command) error {
    force := cmd.Bool("force")
    daemon := NewDaemon(cfg)
    return daemon.Stop(force)  // Business logic receives values
}
```
