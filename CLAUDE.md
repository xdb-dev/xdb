# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Testing

```bash
# Build all packages
go build ./...

# Run all tests with race detection and coverage
make test
# Equivalent: go test -race -timeout=5m -covermode=atomic -coverprofile=coverage.out ./...

# Run tests for specific package
go test ./core
go test ./driver/xdbsqlite

# Run tests for a single function
go test -run TestSpecificFunction ./core
```

### Code Quality and Formatting

```bash
# Run all quality checks (format, vet, lint, tidy imports)
make check

# Individual quality commands
make fmt        # go fmt
make vet        # go vet
make lint       # revive linting
make imports    # gci import formatting
make tidy       # go mod tidy

# Setup project and dependencies
make setup
```

### Coverage Reports

```bash
# Generate coverage report
make coverage

# Generate HTML coverage report and open in browser
make report
```

## Core Architecture

XDB is a tuple-based database abstraction library with a layered architecture:

### 1. Core Data Model (`core/` package)

- **Tuple**: Fundamental building block combining Kind, ID, Attribute Name, and Value
- **Record**: Collection of tuples sharing the same kind and ID (similar to database rows)
- **Key**: Unique reference to a record or tuple attribute
- **Value**: Typed value container with casting methods
- **Schema**: Domain model definitions and constraints

### 2. Driver Layer (`driver/` package)

Bridges XDB's tuple model to specific database backends:

- **Interfaces**: `TupleReader/Writer`, `RecordReader/Writer`, `SchemaReader/Writer`
- **Implementations**:
  - `xdbmemory/`: In-memory driver for testing
  - `xdbsqlite/`: SQLite driver with migrations
  - `xdbredis/`: Redis driver for key-value storage

### 3. Encoding Layer (`encoding/` package)

Converts between XDB types and various formats:

- `xdbjson/`: JSON encoding/decoding
- `xdbstruct/`: Go struct ↔ XDB record conversion
- `xdbproto/`: Protocol Buffer support

### 4. Codec Layer (`codec/` package)

Low-level serialization interfaces for key-value storage:

- `KeyValueCodec`: Interface for marshaling/unmarshaling keys and values
- `msgpack/`: MessagePack implementation

### 5. Supporting Packages

- `registry/`: Schema management and retrieval
- `x/`: Utility functions for grouping, mapping, filtering
- `tests/`: Shared test helpers and fixtures

## Package Structure Guidelines

### File Organization

- Each type or closely related set of types should have its own file
- Test files use `_test.go` suffix and are co-located with source
- Example functions go in `examples_test.go` and are named `ExampleXxx`
- Package documentation goes in `doc.go`

### Driver Implementation

- Each driver lives in its own subdirectory under `driver/`
- Drivers implement capability interfaces (not all drivers support all features)
- Use the existing `xdbmemory` driver as a reference for new implementations

### Encoding Implementation

- Each format lives in its own subdirectory under `encoding/`
- Follow the pattern of existing encoders for consistency

## Code Style Requirements

### Import Organization

Group imports with blank lines between groups:

1. Standard library
2. Third-party packages
3. Local packages (github.com/xdb-dev/xdb/...)

### Error Handling

- Return errors as the last return value
- Use `fmt.Errorf` for error creation with context
- Return early on errors using `if` statements
- Do not use generic error names; be descriptive

### Naming Conventions

- Use CamelCase for exported names, mixedCaps for unexported
- Capitalize acronyms (ID, URL, API)
- Avoid underscores in names
- Use short, meaningful receiver names

### Testing

- Use table-driven tests for functions with multiple cases
- Place test functions in `package_test` form when testing public APIs
- Use `testify/assert` for assertions
- Example tests must have exact `// Output:` comments

## Key Interfaces to Understand

### Driver Interfaces (`driver/driver.go`)

```go
type TupleReader interface {
    GetTuples(ctx context.Context, keys []*core.Key) ([]*core.Tuple, []*core.Key, error)
}

type TupleWriter interface {
    PutTuples(ctx context.Context, tuples []*core.Tuple) error
    DeleteTuples(ctx context.Context, keys []*core.Key) error
}
```

### Codec Interface (`codec/codec.go`)

```go
type KeyValueCodec interface {
    MarshalKey(key *core.Key) ([]byte, error)
    UnmarshalKey(data []byte) (*core.Key, error)
    MarshalValue(value *core.Value) ([]byte, error)
    UnmarshalValue(data []byte) (*core.Value, error)
}
```

## Important Notes

- All exported functions and types must have GoDoc comments
- Use context.Context as the first parameter for functions that need it
- Linting is enforced via `revive` with configuration in `revive.toml`
