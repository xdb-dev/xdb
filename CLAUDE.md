# CLAUDE.md

This file provides guidance for working with code in this repository.

## Development Commands

### Build and Test

- `make test` - Run all tests with race detection and coverage
- `make check` - Run all code quality checks (formatting, vetting, linting)
- `make fmt` - Format code with `go fmt`
- `make lint` - Run revive linter
- `make vet` - Run `go vet`
- `make imports` - Format imports with gci
- `make tidy` - Run `go mod tidy`
- `make setup` - Initial project setup

### Coverage Reports

- `make coverage` - Generate coverage report
- `make report` - Generate HTML coverage report and open in browser

### Single Test Execution

- `go test ./core` - Run tests for core package
- `go test -run TestSpecificTest ./package` - Run specific test

### Running the Server

- `cd ./cmd/xdb && go run *.go server` - Start the XDB HTTP server
- The server runs on port 8080 by default
- Use Bruno collections in `bruno/` directory for testing API endpoints

## Core Architecture

XDB is a tuple-based database abstraction library with a layered architecture:

### 1. Core Data Model (`core/` package)

- **Tuple**: Fundamental building block combining ID, Attribute, and Value
- **Record**: Collection of tuples sharing the same ID (similar to database rows)
- **Key**: Unique reference to a record or tuple
- **Value**: Typed value container with casting methods
- **ID**: Hierarchical identifier represented as string slices
- **Attr**: Attribute names supporting nested structures

### 2. Driver Layer (`driver/` package)

Bridges XDB's tuple model to specific database backends:

- **Interfaces**: `TupleReader/Writer`, `RecordReader/Writer`
- **Current Implementations**:
  - `xdbmemory/`: In-memory driver for testing
  - `xdbsqlite/`: SQLite driver with migrations
  - `xdbredis/`: Redis driver for key-value storage

### 3. API Layer (`api/` package)

HTTP API layer providing RESTful endpoints for tuple operations:

- **TupleAPI**: Handles tuple CRUD operations (Get, Put, Delete)
- **Types**: API-specific types for requests and responses
- **Generic EndpointFunc**: Type-safe endpoint handlers with request/response types

### 4. Encoding Layer (`encoding/` package)

Converts between XDB types and various formats:

- `xdbjson/`: JSON encoding/decoding
- `xdbstruct/`: Go struct ↔ XDB record conversion
- `xdbproto/`: Protocol Buffer support

### 5. Codec Layer (`codec/` package)

Low-level serialization interfaces for key-value storage:

- `codec.go`: Core `KeyValueCodec` interface for marshaling/unmarshaling
- `json/`: JSON codec implementation
- `msgpack/`: MessagePack implementation

### 6. Command Line Application (`cmd/xdb/` package)

CLI application for running XDB server:

- **server**: Starts HTTP server with tuple API endpoints
- **config**: Configuration management for server settings
- Uses SQLite as the default storage backend

### 7. Supporting Packages

- `x/`: Utility functions for grouping, mapping, filtering
- `tests/`: Shared test helpers and fixtures
- `examples/`: Self-contained example applications
- `bruno/`: Bruno API client collections for testing endpoints

## Project Structure

XDB uses a multi-module structure where some packages have their own `go.mod` files:

- Root module: Core packages (`core/`, `driver/`, `x/`, `tests/`)
- `cmd/xdb/`: CLI application module
- `driver/xdbsqlite/`: SQLite driver module
- `driver/xdbredis/`: Redis driver module
- `codec/msgpack/`: MessagePack codec module
- `encoding/xdbjson/`: JSON encoding module
- `encoding/xdbproto/`: Protocol Buffer encoding module

The Makefile automatically runs commands across all modules in the repository, ensuring consistent formatting, testing, and linting across the entire codebase.

## Package Structure Guidelines

### File Organization

- Each type or closely related set of types should have its own file
- Test files use `_test.go` suffix and are co-located with source
- Example functions go in `examples_test.go` and are named `ExampleXxx`
- Package documentation goes in `doc.go`
- Use package `doc.go` for understaning the package

### Driver Implementation

- Each driver lives in its own subdirectory under `driver/`
- Drivers implement capability interfaces (not all drivers support all features)
- Use the existing `xdbmemory` driver as a reference for new implementations

### Encoding Implementation

- Each format lives in its own subdirectory under `encoding/`
- Follow the pattern of existing encoders for consistency

### API Implementation

- API endpoints are defined in `api/` package
- Each API type (Tuple, Record, etc.) has its own file with handler implementations
- Request and response types are defined in `types.go`
- Use generic `EndpointFunc[Req, Res]` type for type-safe endpoint handlers
- API types should be separate from core types to allow independent evolution

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

## Important Notes

- All exported functions and types must have GoDoc comments
- Use context.Context as the first parameter for functions that need it
- Linting is enforced via `revive` with configuration in `revive.toml`
- Make sure to keep GoDoc comments and respective package's`doc.go` up to date
